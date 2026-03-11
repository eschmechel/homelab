package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Alert struct {
	Title   string
	Message string
	Level   AlertLevel
}

type AlertLevel string

const (
	AlertInfo    AlertLevel = "info"
	AlertSuccess AlertLevel = "success"
	AlertWarning AlertLevel = "warning"
	AlertError   AlertLevel = "error"
)

type TelegramConfig struct {
	BotToken string
	ChatID   string
}

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type TelegramClient struct {
	config TelegramConfig
	client *http.Client
}

func NewTelegramClient(botToken, chatID string) *TelegramClient {
	return &TelegramClient{
		config: TelegramConfig{
			BotToken: botToken,
			ChatID:   chatID,
		},
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func NewTelegramClientFromEnv() (*TelegramClient, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}
	if chatID == "" {
		return nil, fmt.Errorf("TELEGRAM_CHAT_ID not set")
	}

	return NewTelegramClient(token, chatID), nil
}

func (t *TelegramClient) Send(alert Alert) error {
	emoji := ""
	switch alert.Level {
	case AlertSuccess:
		emoji = "✅"
	case AlertWarning:
		emoji = "⚠️"
	case AlertError:
		emoji = "❌"
	default:
		emoji = "ℹ️"
	}

	message := fmt.Sprintf("%s *%s*\n\n%s", emoji, alert.Title, alert.Message)
	return t.sendMessage(message)
}

func (t *TelegramClient) sendMessage(text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.config.BotToken)

	msg := TelegramMessage{
		ChatID:    t.config.ChatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

func SendRSSUpdate(client *TelegramClient, totalArticles int) error {
	if client == nil {
		return nil
	}
	alert := Alert{
		Title:   "RSS Update",
		Message: fmt.Sprintf("%d new articles fetched and saved to vault", totalArticles),
		Level:   AlertInfo,
	}
	return client.Send(alert)
}

func SendJobAlert(client *TelegramClient, company, status string) error {
	if client == nil {
		return nil
	}
	alert := Alert{
		Title:   "Job Alert",
		Message: fmt.Sprintf("%s: %s", company, status),
		Level:   AlertInfo,
	}
	return client.Send(alert)
}

func SendDailySummary(client *TelegramClient, tasksCompleted int, focusAreas string) error {
	if client == nil {
		return nil
	}
	alert := Alert{
		Title:   "Daily Summary",
		Message: fmt.Sprintf("Tasks completed: %d\nFocus areas: %s", tasksCompleted, focusAreas),
		Level:   AlertSuccess,
	}
	return client.Send(alert)
}

func SendError(client *TelegramClient, service, errMsg string) error {
	if client == nil {
		return nil
	}
	alert := Alert{
		Title:   fmt.Sprintf("%s Error", service),
		Message: errMsg,
		Level:   AlertError,
	}
	return client.Send(alert)
}
