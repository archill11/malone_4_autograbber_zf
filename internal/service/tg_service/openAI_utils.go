package tg_service

import (
	"bytes"
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

	var mess bytes.Buffer
	mess.WriteString("Ты ведешь телеграм-канал. Твоя задача уникализировать контент, чтобы посты не были друг на друга похожи.")
	mess.WriteString("Я буду отправлять текст, а ты его перефразируй, дели на абзацы, добавляй смайлы, чтобы получить максимально уникальный пост.\n")
	mess.WriteString("Если встретишь слова типо @lichka не нужно их изменять вообще, оставь их как есть, и не нужно добавлять никах markdown форматирований типо звездочек и прочего.\n")
	mess.WriteString("Нужен только итоговый текст, без предисловий и твоих предложений сделать что-либо ещё в конце.\n")
	mess.WriteString(fmt.Sprintf("текст ниже:\n\n%v", postText))
	inputPrompt := mess.String()

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