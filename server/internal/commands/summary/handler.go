package summary

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// TODO: move summary logic into service
type Handler struct {
	client  *pluginapi.Client
	service summarizer
}

const summaryTrigger = "summary"

func New(client *pluginapi.Client, service summarizer) *Handler {
	err := client.SlashCommand.Register(&model.Command{
		Trigger:          summaryTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: "Generate a summary of current channel or thread",
		AutoCompleteHint: "[thread|channel]",
		AutocompleteData: model.NewAutocompleteData(summaryTrigger, "[thread|channel]", "Generate summary of current thread or channel"),
	})
	if err != nil {
		client.Log.Error("Failed to register summary command", "error", err)
	}
	return &Handler{
		client:  client,
		service: service,
	}
}

func (h Handler) Handle(args *model.CommandArgs) (*model.CommandResponse, error) {
	ctx := context.Background()
	trigger := strings.TrimPrefix(strings.Fields(args.Command)[0], "/")
	if trigger != summaryTrigger {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("Unknown command: %s", args.Command),
		}, nil
	}

	fields := strings.Fields(args.Command)
	summaryType := "thread" // default to thread
	if len(fields) > 1 {
		summaryType = fields[1]
	}

	var postList *model.PostList
	var err error
	var summaryTitle string

	switch summaryType {
	case "thread":
		if args.RootId == "" {
			return &model.CommandResponse{
				ResponseType: model.CommandResponseTypeEphemeral,
				Text:         "This command must be used in a thread. Reply to a message first, or use `/summary channel` to summarize the entire channel.",
			}, nil
		}
		postList, err = h.client.Post.GetPostThread(args.RootId)
		summaryTitle = "Thread Summary:"
	case "channel":
		postList, err = h.client.Post.GetPostsForChannel(args.ChannelId, 0, 50)
		summaryTitle = "Channel Summary (last 50 messages):"
	default:
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Usage: /summary [thread|channel]",
		}, nil
	}

	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("Failed to get posts: %v", err),
		}, nil
	}

	summary := h.generateSummary(ctx, postList)
	if summary == "" {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Failed to generate summary.",
		}, nil
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         fmt.Sprintf("**%s**\n%s", summaryTitle, summary),
	}, nil
}

func (c *Handler) generateSummary(ctx context.Context, postList *model.PostList) string {
	if postList == nil || len(postList.Posts) == 0 {
		return "No messages found to summarize."
	}

	summary, err := c.service.GenerateSummary(ctx, postList.ToSlice())
	if err != nil {
		c.client.Log.Error("failed to generate summary: %w", err)
		return ""
	}
	return summary
}
