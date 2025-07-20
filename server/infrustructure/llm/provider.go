package llm

import (
	"context"
)

type Provider interface {
	// Generate creates a response from the given prompt.
	// Note: This method should be used with exponential backoff retry logic
	// to handle temporary failures, rate limits, and network issues from LLM providers.
	Generate(ctx context.Context, prompt string) (string, error)
	HealthCheck(ctx context.Context) error
}
