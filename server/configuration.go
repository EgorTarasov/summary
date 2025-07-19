package main

import (
	"reflect"

	"github.com/pkg/errors"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type configuration struct {
	// TODO: create seperate structs and functions for different model providers
	// LLM Provider Configuration
	LLMProvider   string `json:"llm_provider"`   // "ollama", "openai"

	// Ollama Configuration
	OllamaURL     string `json:"ollama_url"`     // e.g., "http://localhost:11434"
	OllamaModel   string `json:"ollama_model"`   // e.g., "llama2", "mistral", "codellama"

	// OpenAI Configuration
	OpenAIAPIKey  string `json:"openai_api_key"`
	OpenAIModel   string `json:"openai_model"`   // e.g., "gpt-3.5-turbo", "gpt-4"
	OpenAIBaseURL string `json:"openai_base_url"` // For OpenAI-compatible APIs

	// Summary Configuration
	MaxTokens        int     `json:"max_tokens"`         // Maximum tokens for summary
	Temperature      float32 `json:"temperature"`        // LLM temperature (0.0-1.0)
	SummaryLanguage  string  `json:"summary_language"`   // "en", "ru", "auto"
	MaxMessages      int     `json:"max_messages"`       // Max messages to process per request

	// Feature Flags
	EnableChannelSummary bool `json:"enable_channel_summary"`
	EnableThreadSummary  bool `json:"enable_thread_summary"`
	EnableCaching        bool `json:"enable_caching"`

	// Advanced Settings
	RequestTimeout   int    `json:"request_timeout"`    // Timeout in seconds
	SystemPrompt     string `json:"system_prompt"`      // Custom system prompt
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *configuration) Clone() *configuration {
	var clone = *c
	return &clone
}

// IsValid checks if the configuration is valid
func (c *configuration) IsValid() error {
	if c.LLMProvider == "" {
		return errors.New("LLM provider must be specified")
	}

	switch c.LLMProvider {
	case "ollama":
		if c.OllamaURL == "" {
			return errors.New("Ollama URL must be specified when using Ollama provider")
		}
		if c.OllamaModel == "" {
			return errors.New("Ollama model must be specified when using Ollama provider")
		}
	case "openai":
		if c.OpenAIAPIKey == "" {
			return errors.New("OpenAI API key must be specified when using OpenAI provider")
		}
		if c.OpenAIModel == "" {
			return errors.New("OpenAI model must be specified when using OpenAI provider")
		}
	default:
		return errors.Errorf("unsupported LLM provider: %s", c.LLMProvider)
	}

	if c.MaxTokens <= 0 {
		return errors.New("max_tokens must be greater than 0")
	}

	if c.Temperature < 0 || c.Temperature > 1 {
		return errors.New("temperature must be between 0.0 and 1.0")
	}

	if c.MaxMessages <= 0 {
		return errors.New("max_messages must be greater than 0")
	}

	if c.RequestTimeout <= 0 {
		return errors.New("request_timeout must be greater than 0")
	}

	return nil
}

// SetDefaults sets default values for configuration
func (c *configuration) SetDefaults() {
	if c.LLMProvider == "" {
		c.LLMProvider = "ollama"
	}

	if c.OllamaURL == "" {
		c.OllamaURL = "http://localhost:11434"
	}

	if c.OllamaModel == "" {
		c.OllamaModel = "llama2"
	}

	if c.OpenAIModel == "" {
		c.OpenAIModel = "gpt-3.5-turbo"
	}

	if c.OpenAIBaseURL == "" {
		c.OpenAIBaseURL = "http://localhost:11434/v1"
	}

	if c.MaxTokens == 0 {
		c.MaxTokens = 1000
	}

	if c.Temperature == 0 {
		c.Temperature = 0.3
	}

	if c.SummaryLanguage == "" {
		c.SummaryLanguage = "auto"
	}

	if c.MaxMessages == 0 {
		c.MaxMessages = 50
	}

	if c.RequestTimeout == 0 {
		c.RequestTimeout = 30
	}

	if c.SystemPrompt == "" {
		c.SystemPrompt = "You are a helpful assistant that creates concise summaries of chat conversations. Focus on key points, decisions, and action items."
	}

	// Enable features by default
	c.EnableChannelSummary = true
	c.EnableThreadSummary = true
	c.EnableCaching = false
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	var configuration = new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	// Set defaults for any missing values
	configuration.SetDefaults()

	// Validate the configuration
	if err := configuration.IsValid(); err != nil {
		return errors.Wrap(err, "invalid plugin configuration")
	}

	p.setConfiguration(configuration)

	return nil
}
