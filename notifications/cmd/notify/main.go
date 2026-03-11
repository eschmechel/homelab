package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	JobAlertsFile = "/tmp/job_alerts_today.json"
)

type AlertLevel string

const (
	AlertInfo    AlertLevel = "info"
	AlertSuccess AlertLevel = "success"
	AlertWarning AlertLevel = "warning"
	AlertError   AlertLevel = "error"
)

type JobAlert struct {
	Company string `json:"company"`
	Status  string `json:"status"`
	Time    string `json:"time"`
}

type Config struct {
	TelegramBotToken string `json:"telegram_bot_token"`
	TelegramChatID   string `json:"telegram_chat_id"`
}

func loadConfig() Config {
	return Config{
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func sendTelegram(config Config, title, message string, level AlertLevel) error {
	if config.TelegramBotToken == "" || config.TelegramChatID == "" {
		return fmt.Errorf("Telegram credentials not configured")
	}

	emoji := "ℹ️"
	switch level {
	case AlertSuccess:
		emoji = "✅"
	case AlertWarning:
		emoji = "⚠️"
	case AlertError:
		emoji = "❌"
	}

	text := fmt.Sprintf("%s *%s*\n\n%s", emoji, title, message)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.TelegramBotToken)

	msg := map[string]interface{}{
		"chat_id":    config.TelegramChatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	jsonData, _ := json.Marshal(msg)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Telegram API error: %s", string(body))
	}

	return nil
}

func trackJobAlert(company, status string) error {
	var alerts []JobAlert

	if data, err := os.ReadFile(JobAlertsFile); err == nil {
		json.Unmarshal(data, &alerts)
	}

	alerts = append(alerts, JobAlert{
		Company: company,
		Status:  status,
		Time:    time.Now().Format(time.RFC3339),
	})

	dir := "/tmp"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(alerts)
	if err != nil {
		return err
	}
	return os.WriteFile(JobAlertsFile, data, 0644)
}

func getDailyJobSummary() (string, error) {
	var alerts []JobAlert

	data, err := os.ReadFile(JobAlertsFile)
	if err != nil {
		return "No job alerts today", nil
	}

	if err := json.Unmarshal(data, &alerts); err != nil {
		return "", err
	}

	if len(alerts) == 0 {
		return "No job alerts today", nil
	}

	summary := fmt.Sprintf("📊 *Daily Job Summary*\n\n*Total:* %d alerts\n\n", len(alerts))
	for _, alert := range alerts {
		summary += fmt.Sprintf("• *%s:* %s\n", alert.Company, alert.Status)
	}

	return summary, nil
}

func clearJobAlerts() error {
	return os.WriteFile(JobAlertsFile, []byte("[]"), 0644)
}

func handleJobCommand(config Config, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("Usage: notify job <company> <status>")
	}

	company := args[0]
	status := args[1]

	// Always send notification for job alerts
	if err := sendTelegram(config, "Job Alert", fmt.Sprintf("%s: %s", company, status), AlertInfo); err != nil {
		return err
	}

	// Track for daily summary
	if err := trackJobAlert(company, status); err != nil {
		return err
	}

	fmt.Printf("Job alert sent and tracked: %s - %s\n", company, status)
	return nil
}

func handleRSSCommand(config Config, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("Usage: notify rss <count>")
	}

	count := args[0]

	if err := sendTelegram(config, "RSS Update", fmt.Sprintf("%s new articles fetched and saved to vault", count), AlertInfo); err != nil {
		return err
	}

	fmt.Printf("RSS notification sent: %s articles\n", count)
	return nil
}

func handleDailySummaryCommand(config Config) error {
	summary, err := getDailyJobSummary()
	if err != nil {
		return err
	}

	if err := sendTelegram(config, "Daily Summary", summary, AlertSuccess); err != nil {
		return err
	}

	// Clear after sending
	clearJobAlerts()

	fmt.Printf("Daily summary sent\n")
	return nil
}

func handleGenericCommand(config Config, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("Usage: notify generic <title> <message> [level]")
	}

	title := args[0]
	message := args[1]
	level := AlertInfo
	if len(args) > 2 {
		level = AlertLevel(args[2])
	}

	if err := sendTelegram(config, title, message, level); err != nil {
		return err
	}

	fmt.Printf("Notification sent: %s\n", title)
	return nil
}

func main() {
	config := loadConfig()

	if config.TelegramBotToken == "" {
		fmt.Fprintln(os.Stderr, "Error: TELEGRAM_BOT_TOKEN not set")
		os.Exit(1)
	}
	if config.TelegramChatID == "" {
		fmt.Fprintln(os.Stderr, "Error: TELEGRAM_CHAT_ID not set")
		os.Exit(1)
	}

	command := flag.String("cmd", "", "Command: job, rss, daily-summary, generic")
	flag.Parse()

	args := flag.Args()

	if *command != "" {
		args = append([]string{*command}, args...)
	}

	if len(args) < 1 {
		fmt.Println("Usage: notify <command> [args]")
		fmt.Println("")
		fmt.Println("Commands:")
		fmt.Println("  job <company> <status>              - Send job alert (always notifies)")
		fmt.Println("  rss <count>                         - Send RSS update notification")
		fmt.Println("  daily-summary                       - Send daily job summary")
		fmt.Println("  generic <title> <message> [level]  - Send generic notification")
		os.Exit(1)
	}

	cmd := strings.ToLower(args[0])
	var err error

	switch cmd {
	case "job":
		err = handleJobCommand(config, args[1:])
	case "rss":
		err = handleRSSCommand(config, args[1:])
	case "daily-summary", "dailysummary":
		err = handleDailySummaryCommand(config)
	case "generic":
		err = handleGenericCommand(config, args[1:])
	default:
		err = fmt.Errorf("Unknown command: %s", cmd)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
