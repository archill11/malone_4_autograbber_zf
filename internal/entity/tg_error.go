package entity

type TgError struct {
	BotId          int    `json:"bot_id"`
	BotToken       string `json:"bot_token"`
	BotUsername    string `json:"bot_username"`
	BotChId        int    `json:"bot_ch_id"`
	ErrDescription string `json:"err_description"`
	ErrCount       int    `json:"err_count"`
}
