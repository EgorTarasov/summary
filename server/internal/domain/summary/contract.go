package summary

import (
	"context"

	"github.com/mattermost/mattermost/server/public/model"
)

type (
	llm interface {
		Generate(ctx context.Context, prompt string) (string, error)
	}
	userProvider interface {
		Get(userID string) (*model.User, error)
	}
)
