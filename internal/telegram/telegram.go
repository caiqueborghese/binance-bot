package telegram

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
)

func SendMessage(text string) error {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chatID == "" {
		return fmt.Errorf("TOKEN ou CHAT_ID ausente")
	}

	msg := url.QueryEscape(text)
	endpoint := fmt.Sprintf(
		"https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s",
		token, chatID, msg,
	)

	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
