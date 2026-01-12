package tg_service

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"go.uber.org/zap"
)

func (srv *TgService) CreateGptTextOpenAI(postText string) (string, error) {
	client := openai.NewClient(
		option.WithAPIKey(srv.Cfg.OpenAiAPIToken),
	)

	inputPrompt := fmt.Sprintf(
		"Ты ведешь телеграм-канал. Твоя задача уникализировать контент, чтобы посты не были друг на друга похожи. Я буду отправлять текст, а ты его перефразируй, дели на абзацы, добавляй смайлы, чтобы получить максимально уникальный пост.\nтекст ниже:\n\n%v",
		postText,
	)

	resp, err := client.Responses.New(context.Background(), responses.ResponseNewParams{
		Model: responses.ChatModelGPT4_1Mini,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(inputPrompt),
		},
	})
	if err != nil {
		return "", err
	}

	finalText := resp.OutputText()
	if finalText == "" {
		srv.l.Error("CreateGptTextOpenAI err: finalText is empty", zap.Any("resp", resp), zap.Any("inputPrompt", inputPrompt))
		return "", fmt.Errorf("CreateGptTextOpenAI err: finalText is empty")
	}

	return finalText, nil
}