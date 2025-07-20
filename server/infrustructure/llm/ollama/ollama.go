package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EgorTarasov/summary/server/infrustructure/ptr"
	"github.com/ollama/ollama/api"
)

// TODO: add system metrics such as tokens / second, len of prompts, time of response and any valueble info
var avaliableModels map[string]struct{} = map[string]struct{}{
	"gemma3:12b": {},
}

const (
	defaultHost           = "http://localhost:11434"
	defaultModel          = "gemma3:12b"
	defaultContextSize    = 64000
	systemPromptMaxLength = 3200
	minContextSize        = 1024
	maxContextSize        = 128000
)

type Option func(c *config) (*config, error)

func WithHost(host string) Option {
	return func(c *config) (*config, error) {
		if strings.TrimSpace(host) == "" {
			return c, fmt.Errorf("host cannot be empty")
		}

		baseURL, err := url.Parse(host)
		if err != nil {
			return c, fmt.Errorf("invalid baseURL for ollama: %w", err)
		}

		if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
			return c, fmt.Errorf("invalid URL scheme '%s': must be http or https", baseURL.Scheme)
		}

		if baseURL.Host == "" {
			return c, fmt.Errorf("invalid URL: host cannot be empty")
		}

		c.baseURL = baseURL
		return c, nil
	}
}

func WithModel(model string) func(c *config) (*config, error) {
	return func(c *config) (*config, error) {
		if _, ok := avaliableModels[model]; !ok {
			return c, fmt.Errorf("unsupported model: %v", model)
		}
		c.model = model
		return c, nil
	}
}

func WithSystemPrompt(propmt string) func(c *config) (*config, error) {
	return func(c *config) (*config, error) {
		if len(propmt) > systemPromptMaxLength {
			return c, fmt.Errorf("system prompt must not exceed %v: given prompt is: %v", systemPromptMaxLength, len(propmt))
		}
		c.systemPrompt = propmt
		return c, nil
	}
}

func WithContextSize(size int) Option {
	return func(c *config) (*config, error) {
		if size < minContextSize {
			return c, fmt.Errorf("context size must be at least %d, got %d", minContextSize, size)
		}
		if size > maxContextSize {
			return c, fmt.Errorf("context size must not exceed %d, got %d", maxContextSize, size)
		}
		c.contextSize = size
		return c, nil
	}
}

type config struct {
	baseURL      *url.URL
	model        string
	systemPrompt string
	contextSize  int
}

func (c *config) validate() error {
	if c.baseURL == nil {
		return fmt.Errorf("baseURL is required")
	}

	if c.model == "" {
		return fmt.Errorf("model is required")
	}

	if c.contextSize < minContextSize || c.contextSize > maxContextSize {
		return fmt.Errorf("context size %d is out of valid range [%d, %d]",
			c.contextSize, minContextSize, maxContextSize)
	}

	return nil
}

func (c *config) setDefaults() error {
	if c.baseURL == nil {
		defaultURL, err := url.Parse(defaultHost)
		if err != nil {
			return fmt.Errorf("failed to parse default host: %w", err)
		}
		c.baseURL = defaultURL
	}

	if c.model == "" {
		c.model = defaultModel
	}

	if c.contextSize == 0 {
		c.contextSize = defaultContextSize
	}

	return nil
}

type OllamaProvider struct {
	cfg *config
	api *api.Client
}

func New(options ...Option) (*OllamaProvider, error) {
	cfg := &config{}

	for _, opt := range options {
		var err error
		cfg, err = opt(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	if err := cfg.setDefaults(); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	client := api.NewClient(cfg.baseURL, http.DefaultClient)

	return &OllamaProvider{
		cfg: cfg,
		api: client,
	}, nil
}

func (p OllamaProvider) HealthCheck(ctx context.Context) error {
	_, err := p.api.List(ctx)
	if err != nil {
		return fmt.Errorf("ollama server is unavaliable: %w", err)
	}
	return nil
}

func (p OllamaProvider) Generate(ctx context.Context, prompt string) (string, error) {
	in := &api.GenerateRequest{
		Model:    p.cfg.model,
		Prompt:   prompt,
		Suffix:   "",
		System:   "",
		Template: "",
		Context:  []int{},
		Stream:   ptr.To(false),
		Raw:      false,
		Format:   json.RawMessage{},
		KeepAlive: &api.Duration{
			Duration: time.Hour * 1,
		},
		Images:  []api.ImageData{},
		Options: map[string]any{},
		Think:   ptr.To(false),
	}
	resp := strings.Builder{}
	err := p.api.Generate(ctx, in, func(gr api.GenerateResponse) error {
		resp.WriteString(gr.Response)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to send request")
	}
	return resp.String(), nil
}
