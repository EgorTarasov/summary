package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/EgorTarasov/summary/server/infrustructure/llm/ollama"
	summaryCommand "github.com/EgorTarasov/summary/server/internal/commands/summary"
	"github.com/EgorTarasov/summary/server/internal/domain/summary"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type Command interface {
	Handle(args *model.CommandArgs) (*model.CommandResponse, error)
}

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// // kvstore is the client used to read/write KV records for this plugin.
	// kvstore kvstore.KVStore

	// client is the Mattermost server API client.
	// client *pluginapi.Client

	// commandClient is the client used to register and execute slash commands.
	commandClient Command

	// backgroundJob *cluster.Job

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin will be deactivated.
func (p *Plugin) OnActivate() error {
	client := pluginapi.NewClient(p.API, p.Driver)
	client.Log.Info("Plugin activation started")

	c := p.getConfiguration()
	if c == nil {
		client.Log.Error("Failed to get plugin configuration")
		return fmt.Errorf("configuration is not available")
	}

	c.SetDefaults()

	client.Log.Info("ollama configuration", "model", c.OllamaModel, "baseURL", c.OllamaURL, "provider", c.LLMProvider)

	if err := c.IsValid(); err != nil {
		client.Log.Error("Invalid plugin configuration", "error", err.Error())
		return fmt.Errorf("invalid configuration: %w", err)
	}

	ollamaProvider, err := ollama.New(
		ollama.WithHost(c.OllamaURL),
		ollama.WithModel(c.OllamaModel),
	)
	if err != nil {
		client.Log.Error("Failed to initialize Ollama provider", "error", err.Error())
		return fmt.Errorf("failed to init ollama: %w", err)
	}

	client.Log.Info("Ollama provider initialized successfully")

	summaryService := summary.NewService(ollamaProvider, &client.User)

	summaryHandler := summaryCommand.New(client, summaryService)
	p.commandClient = summaryHandler

	client.Log.Info("Plugin activated successfully")

	// p.kvstore = kvstore.NewKVStore(p.client)

	// job, err := cluster.Schedule(
	// 	p.API,
	// 	"BackgroundJob",
	// 	cluster.MakeWaitForRoundedInterval(1*time.Hour),
	// 	p.runJob,
	// )
	// if err != nil {
	// 	return errors.Wrap(err, "failed to schedule background job")
	// }

	// p.backgroundJob = job

	return nil
}

// OnDeactivate is invoked when the plugin is deactivated.
func (p *Plugin) OnDeactivate() error {
	// if p.backgroundJob != nil {
	// 	if err := p.backgroundJob.Close(); err != nil {
	// 		p.API.LogError("Failed to close background job", "err", err)
	// 	}
	// }
	return nil
}

// This will execute the commands that were registered in the NewCommandHandler function.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if p.commandClient == nil {
		p.API.LogError("Command client is not initialized")
		return nil, model.NewAppError("ExecuteCommand", "plugin.command.not_initialized", nil, "command client not initialized", http.StatusInternalServerError)
	}

	response, err := p.commandClient.Handle(args)
	if err != nil {
		p.API.LogError("Failed to execute command", "error", err.Error(), "command", args.Command)
		return nil, model.NewAppError("ExecuteCommand", "plugin.command.execute_command.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	return response, nil
}

// MessageWillBePosted is called before a message is posted by a user.
func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
	// Add any message preprocessing logic here if needed
	return post, ""
}

// MessageHasBeenPosted is called after a message has been posted by a user.
func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {
	// Add any message post-processing logic here if needed
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
