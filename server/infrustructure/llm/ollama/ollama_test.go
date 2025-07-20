package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		options     []Option
		expectError bool
		errorMsg    string
	}{
		{
			name:        "should_create_provider_with_default_config",
			options:     []Option{},
			expectError: false,
		},
		{
			name: "should_create_provider_with_valid_host",
			options: []Option{
				WithHost("http://localhost:11434"),
			},
			expectError: false,
		},
		{
			name: "should_create_provider_with_valid_model",
			options: []Option{
				WithModel("gemma3:12b"),
			},
			expectError: false,
		},
		{
			name: "should_create_provider_with_valid_system_prompt",
			options: []Option{
				WithSystemPrompt("You are a helpful assistant"),
			},
			expectError: false,
		},
		{
			name: "should_create_provider_with_all_valid_options",
			options: []Option{
				WithHost("http://localhost:11434"),
				WithModel("gemma3:12b"),
				WithSystemPrompt("You are a helpful assistant"),
			},
			expectError: false,
		},
		{
			name: "should_fail_with_invalid_host",
			options: []Option{
				WithHost("invalid-url"),
			},
			expectError: true,
		},
		{
			name: "should_fail_with_unsupported_model",
			options: []Option{
				WithModel("invalid-model"),
			},
			expectError: true,
			errorMsg:    "unsupported model",
		},
		{
			name: "should_fail_with_system_prompt_too_long",
			options: []Option{
				WithSystemPrompt(strings.Repeat("a", systemPromptMaxLength+1)),
			},
			expectError: true,
			errorMsg:    "system prompt must not exceed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := New(tt.options...)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, provider)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.NotNil(t, provider.cfg)
				assert.NotNil(t, provider.api)
			}
		})
	}
}

func TestNew_ConfigValidation(t *testing.T) {
	t.Run("should_set_correct_host_URL", func(t *testing.T) {
		expectedURL := "http://localhost:11434"
		provider, err := New(WithHost(expectedURL))

		require.NoError(t, err)
		require.NotNil(t, provider)
		assert.Equal(t, expectedURL, provider.cfg.baseURL.String())
	})

	t.Run("should_set_correct_model", func(t *testing.T) {
		expectedModel := "gemma3:12b"
		provider, err := New(WithModel(expectedModel))

		require.NoError(t, err)
		require.NotNil(t, provider)
		assert.Equal(t, expectedModel, provider.cfg.model)
	})

	t.Run("should_set_correct_system_prompt", func(t *testing.T) {
		expectedPrompt := "You are a helpful assistant"
		provider, err := New(WithSystemPrompt(expectedPrompt))

		require.NoError(t, err)
		require.NotNil(t, provider)
		assert.Equal(t, expectedPrompt, provider.cfg.systemPrompt)
	})
}

func TestOllamaProvider_Generate(t *testing.T) {
	tests := []struct {
		name          string
		prompt        string
		mockResponse  string
		expectError   bool
		errorMsg      string
		setupMockFunc func() *httptest.Server
	}{
		{
			name:         "should_generate_response_successfully",
			prompt:       "Hello, how are you?",
			mockResponse: "I'm doing well, thank you!",
			expectError:  false,
			setupMockFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/generate" {
						response := map[string]interface{}{
							"model":    "gemma3:12b",
							"response": "I'm doing well, thank you!",
							"done":     true,
						}
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(response)
					}
				}))
			},
		},
		{
			name:        "should_handle_empty_prompt",
			prompt:      "",
			expectError: false, // The current implementation allows empty prompts
			setupMockFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/generate" {
						response := map[string]interface{}{
							"model":    "gemma3:12b",
							"response": "Hello! How can I help you?",
							"done":     true,
						}
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(response)
					}
				}))
			},
		},
		{
			name:        "should_handle_streaming_response",
			prompt:      "Tell me a story",
			expectError: false,
			setupMockFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/generate" {
						responses := []map[string]interface{}{
							{"response": "Once ", "done": false},
							{"response": "upon ", "done": false},
							{"response": "a time...", "done": true},
						}

						w.Header().Set("Content-Type", "application/json")
						for _, resp := range responses {
							json.NewEncoder(w).Encode(resp)
						}
					}
				}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupMockFunc()
			defer server.Close()

			serverURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			provider, err := New(
				WithHost(server.URL),
				WithModel("gemma3:12b"),
			)
			require.NoError(t, err)

			provider.api = api.NewClient(serverURL, http.DefaultClient)

			ctx := context.Background()
			result, err := provider.Generate(ctx, tt.prompt)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestOllamaProvider_HealthCheck(t *testing.T) {
	tests := []struct {
		name          string
		expectError   bool
		errorMsg      string
		setupMockFunc func() *httptest.Server
	}{
		{
			name:        "should_pass_health_check_when_server_is_healthy",
			expectError: false,
			setupMockFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/tags" {
						response := map[string]interface{}{
							"models": []interface{}{
								map[string]interface{}{
									"name":        "llama2:latest",
									"size":        3825819519,
									"digest":      "sha256:1a2b3c4d5e6f",
									"modified_at": "2024-01-15T10:30:00Z",
									"details": map[string]interface{}{
										"format":             "gguf",
										"family":             "llama",
										"families":           []string{"llama"},
										"parameter_size":     "7B",
										"quantization_level": "Q4_0",
									},
								},
								map[string]interface{}{
									"name":        "mistral:latest",
									"size":        4109829387,
									"digest":      "sha256:2b3c4d5e6f7a",
									"modified_at": "2024-01-20T14:15:00Z",
								},
							},
						}
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						json.NewEncoder(w).Encode(response)
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
		},
		{
			name:        "should_pass_health_check_with_empty_model_list",
			expectError: false,
			setupMockFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/tags" {
						response := map[string]interface{}{
							"models": []interface{}{},
						}
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						json.NewEncoder(w).Encode(response)
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
		},
		{
			name:        "should_fail_health_check_when_server_returns_500",
			expectError: true,
			errorMsg:    "ollama server is unavaliable", // Match the typo in the actual implementation
			setupMockFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal Server Error"))
				}))
			},
		},
		{
			name:        "should_fail_health_check_when_server_returns_404",
			expectError: true,
			errorMsg:    "ollama server is unavaliable", // Match the typo in the actual implementation
			setupMockFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte("Not Found"))
				}))
			},
		},
		{
			name:        "should_fail_health_check_when_server_returns_unauthorized",
			expectError: true,
			errorMsg:    "ollama server is unavaliable", // Match the typo in the actual implementation
			setupMockFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Unauthorized"))
				}))
			},
		},
		{
			name:        "should_fail_health_check_when_server_returns_bad_gateway",
			expectError: true,
			errorMsg:    "ollama server is unavaliable", // Match the typo in the actual implementation
			setupMockFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusBadGateway)
					w.Write([]byte("Bad Gateway"))
				}))
			},
		},
		{
			name:        "should_fail_health_check_when_server_returns_invalid_json",
			expectError: true,
			errorMsg:    "ollama server is unavaliable", // Match the typo in the actual implementation
			setupMockFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/tags" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{"invalid": json}`)) // Invalid JSON
					}
				}))
			},
		},
		{
			name:        "should_fail_health_check_when_server_times_out",
			expectError: true,
			errorMsg:    "ollama server is unavaliable", // Match the typo in the actual implementation
			setupMockFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Simulate server timeout by sleeping longer than context timeout
					time.Sleep(200 * time.Millisecond)
					response := map[string]interface{}{
						"models": []interface{}{},
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(response)
				}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupMockFunc()
			defer server.Close()

			provider, err := New(
				WithHost(server.URL),
				WithModel("gemma3:12b"), // Use the correct model name
			)
			require.NoError(t, err)

			ctx := context.Background()
			if strings.Contains(tt.name, "times_out") {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}

			err = provider.HealthCheck(ctx)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOllamaProvider_Generate_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		response := map[string]interface{}{
			"model":    "gemma3:12b",
			"response": "This is a delayed response",
			"done":     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider, err := New(
		WithHost(server.URL),
		WithModel("gemma3:12b"),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	result, err := provider.Generate(ctx, "Hello")

	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestOllamaProvider_Generate_RequestStructure(t *testing.T) {
	var capturedRequest *api.GenerateRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			// Capture the request for validation
			var req api.GenerateRequest
			json.NewDecoder(r.Body).Decode(&req)
			capturedRequest = &req

			response := map[string]interface{}{
				"model":    "gemma3:12b",
				"response": "Test response",
				"done":     true,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer server.Close()

	provider, err := New(
		WithHost(server.URL),
		WithModel("gemma3:12b"),
		WithSystemPrompt("You are a test assistant"),
	)
	require.NoError(t, err)

	ctx := context.Background()
	prompt := "Test prompt"

	_, err = provider.Generate(ctx, prompt)
	require.NoError(t, err)

	require.NotNil(t, capturedRequest)
	assert.Equal(t, "gemma3:12b", capturedRequest.Model)
	assert.Equal(t, prompt, capturedRequest.Prompt)
	assert.NotNil(t, capturedRequest.Stream)
	assert.False(t, *capturedRequest.Stream)
	assert.Equal(t, time.Hour, capturedRequest.KeepAlive.Duration)
}
