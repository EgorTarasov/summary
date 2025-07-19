package summary

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)


type Handler struct{
	client *pluginapi.Client
}

const summaryTrigger = "summary"

func New(client *pluginapi.Client) *Handler{
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
	return &Handler{client: client}
}


func (h Handler) Handle(args *model.CommandArgs) (*model.CommandResponse, error){
	trigger := strings.TrimPrefix(strings.Fields(args.Command)[0], "/")
	if trigger != summaryTrigger{
		return &model.CommandResponse{
					ResponseType: model.CommandResponseTypeEphemeral,
					Text:         fmt.Sprintf("Unknown command: %s", args.Command),
				}, nil
	}


	// TODO: define errors
	fields := strings.Fields(args.Command)
	summaryType := "thread" // default to thread
	if len(fields) > 1 {
		summaryType = fields[1]
	}

	switch summaryType {
	case "thread":
		return h.summarizeThread(args), nil
	case "channel":
		return h.summarizeChannel(args), nil
	default:
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Usage: /summary [thread|channel]",
		}, nil
	}
}

func (h *Handler) summarizeThread(args *model.CommandArgs) *model.CommandResponse {
	if args.RootId == "" {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "This command must be used in a thread. Reply to a message first, or use `/summary channel` to summarize the entire channel.",
		}
	}

	// Get all posts in the thread
	postList, err := h.client.Post.GetPostThread(args.RootId)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("Failed to get thread posts: %v", err),
		}
	}

	summary := h.generateSummary(postList)

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         fmt.Sprintf("**Thread Summary:**\n%s", summary),
	}
}

func (h *Handler) summarizeChannel(args *model.CommandArgs) *model.CommandResponse {
	// Get recent posts from the channel (last 50 posts)
	postList, err := h.client.Post.GetPostsForChannel(args.ChannelId, 0, 50)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("Failed to get channel posts: %v", err),
		}
	}

	summary := h.generateSummary(postList)

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         fmt.Sprintf("**Channel Summary (last 50 messages):**\n%s", summary),
	}
}

func (c *Handler) generateSummary(postList *model.PostList) string {
	// TODO: implement model providers in infrustructure
	if postList == nil || len(postList.Posts) == 0 {
		return "No messages found to summarize."
	}

	// Sort posts by creation time
	posts := postList.ToSlice()

	var messages []string
	userMap := make(map[string]int)

	for _, post := range posts {
		if post.DeleteAt == 0 && post.Message != "" { // Only include non-deleted posts with content
			// Get user info
			user, err := c.client.User.Get(post.UserId)
			userName := "Unknown User"
			if err == nil && user != nil {
				userName = user.Username
			}

			userMap[userName]++
			messages = append(messages, fmt.Sprintf("**%s:** %s", userName, post.Message))
		}
	}

	if len(messages) == 0 {
		return "No messages found to summarize."
	}

	// Create participant summary
	participantSummary := "\n**Participants:**\n"
	for user, count := range userMap {
		participantSummary += fmt.Sprintf("- %s (%d messages)\n", user, count)
	}

	// Combine all messages
	allMessages := strings.Join(messages, "\n\n")

	summary := fmt.Sprintf("**Messages (%d total):**\n%s\n%s", 
		len(messages), 
		allMessages,
		participantSummary)

	// Truncate if too long (Mattermost has message limits)
	if len(summary) > 4000 {
		summary = summary[:4000] + "\n\n... (truncated due to length)"
	}

	return summary
}
