package summary

import (
	"context"

	"github.com/mattermost/mattermost/server/public/model"
)

type summarizer interface {
	GenerateSummary(ctx context.Context, posts []*model.Post) (string, error)
}
