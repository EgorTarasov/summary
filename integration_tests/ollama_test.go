//go:build integration

package integrationtests

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/EgorTarasov/summary/server/infrustructure/llm/ollama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultTestHost  = "http://localhost:11434"
	defaultTestModel = "gemma3:12b" // Using a smaller model for faster tests
	testTimeout      = 30 * time.Second
)

// getOllamaHost returns the Ollama host URL from environment or default
func getOllamaHost() string {
	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		return host
	}
	return defaultTestHost
}

// getTestModel returns the test model from environment or default
func getTestModel() string {
	if model := os.Getenv("OLLAMA_TEST_MODEL"); model != "" {
		return model
	}
	return defaultTestModel
}

// skipIfOllamaUnavailable skips the test if Ollama server is not available
func skipIfOllamaUnavailable(t *testing.T, provider *ollama.OllamaProvider) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.HealthCheck(ctx); err != nil {
		t.Skipf("Ollama server is not available: %v", err)
	}
}

func TestOllamaProvider_Integration_HealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		expectError bool
		skipReason  string
	}{
		{
			name:        "should_pass_health_check_with_running_ollama_server",
			host:        getOllamaHost(),
			expectError: false,
		},
		{
			name:        "should_fail_health_check_with_unavailable_server",
			host:        "http://localhost:99999", // Non-existent port
			expectError: true,
			skipReason:  "Testing connection to unavailable server",
		},
		{
			name:        "should_fail_health_check_with_invalid_host",
			host:        "http://invalid-host-that-does-not-exist.local:11434",
			expectError: true,
			skipReason:  "Testing DNS resolution failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Logf("Test info: %s", tt.skipReason)
			}

			provider, err := ollama.New(
				ollama.WithHost(tt.host),
				ollama.WithModel(getTestModel()),
			)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			err = provider.HealthCheck(ctx)

			if tt.expectError {
				assert.Error(t, err)
				t.Logf("Expected error received: %v", err)
			} else {
				if err != nil {
					t.Skipf("Ollama server not available at %s: %v", tt.host, err)
				}
				assert.NoError(t, err)
			}
		})
	}
}

func TestOllamaProvider_Integration_Generate(t *testing.T) {
	provider, err := ollama.New(
		ollama.WithHost(getOllamaHost()),
		ollama.WithModel(getTestModel()),
	)
	require.NoError(t, err)

	// Skip if Ollama is not available
	skipIfOllamaUnavailable(t, provider)

	tests := []struct {
		name             string
		prompt           string
		systemPrompt     string
		expectError      bool
		validateResponse func(t *testing.T, response string)
	}{
		{
			name:   "should_generate_simple_response",
			prompt: "Say hello",
			validateResponse: func(t *testing.T, response string) {
				assert.NotEmpty(t, response)
				assert.True(t, len(response) > 0)
				t.Logf("Generated response: %q", response)
			},
		},
		{
			name:   "should_generate_response_for_math_question",
			prompt: "What is 2 + 2?",
			validateResponse: func(t *testing.T, response string) {
				assert.NotEmpty(t, response)
				assert.Contains(t, strings.ToLower(response), "4")
				t.Logf("Math response: %q", response)
			},
		},
		{
			name:   "should_handle_coding_question",
			prompt: "Write a simple 'Hello, World!' in Python",
			validateResponse: func(t *testing.T, response string) {
				assert.NotEmpty(t, response)
				response_lower := strings.ToLower(response)
				assert.True(t,
					strings.Contains(response_lower, "print") ||
						strings.Contains(response_lower, "hello"),
					"Response should contain Python code or hello: %q", response)
				t.Logf("Coding response: %q", response)
			},
		},
		{
			name:   "should_handle_long_prompt",
			prompt: strings.Repeat("Tell me about artificial intelligence. ", 20),
			validateResponse: func(t *testing.T, response string) {
				assert.NotEmpty(t, response)
				assert.True(t, len(response) > 10) // Expect substantial response
				t.Logf("Long prompt response length: %d chars", len(response))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create provider with system prompt if specified
			testProvider := provider
			if tt.systemPrompt != "" {
				var err error
				testProvider, err = ollama.New(
					ollama.WithHost(getOllamaHost()),
					ollama.WithModel(getTestModel()),
					ollama.WithSystemPrompt(tt.systemPrompt),
				)
				require.NoError(t, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			startTime := time.Now()
			response, err := testProvider.Generate(ctx, tt.prompt)
			duration := time.Since(startTime)

			t.Logf("Generation took: %v", duration)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.validateResponse != nil {
				tt.validateResponse(t, response)
			}
		})
	}
}
