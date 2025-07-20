package summary

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
)

type Service struct {
	llm          llm
	userProvider userProvider
}

func NewService(llm llm, userProvider userProvider) *Service {
	return &Service{
		llm:          llm,
		userProvider: userProvider,
	}
}

func (s Service) GenerateSummary(ctx context.Context, posts []*model.Post) (string, error) {
	conversationText := strings.Builder{}
	for _, post := range posts {
		if post.DeleteAt != 0 && post.Message == "" {
			continue
		}

		user, err := s.userProvider.Get(post.UserId)
		source := "unknown user"
		if err == nil && user != nil {
			source = user.FirstName + " " + user.LastName + " " + user.Position
		}
		conversationText.WriteString(source + ":" + post.Message)
	}
	if conversationText.Len() == 0 {
		return "", fmt.Errorf("no messages")
	}

	prompt := fmt.Sprintf(`Проанализируйте и обобщите следующую командную беседу:

ПЕРЕПИСКА:
%s

Структура резюме:
• **Краткое содержание:** основные темы и направления обсуждения
• **Ключевые решения:** принятые решения и достигнутые договоренности
• **План действий:** поставленные задачи и сроки выполнения
• **Участники:** активные участники и их роль в обсуждении

Используйте четкое форматирование markdown.`, conversationText.String())

	summary, err := s.llm.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	return summary, nil
}
