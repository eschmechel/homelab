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
	Company  string `json:"company"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	URL      string `json:"url"`
	Location string `json:"location"`
	Salary   string `json:"salary"`
	Resume   string `json:"resume"`
	Time     string `json:"time"`
}

type RSSArticle struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Source string `json:"source"`
}

type Config struct {
	TelegramBotToken string
	TelegramChatID   string
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

func trackJobAlert(alert JobAlert) error {
	var alerts []JobAlert

	if data, err := os.ReadFile(JobAlertsFile); err == nil {
		json.Unmarshal(data, &alerts)
	}

	alerts = append(alerts, alert)

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

func getDailyJobSummary() (string, []JobAlert, error) {
	var alerts []JobAlert

	data, err := os.ReadFile(JobAlertsFile)
	if err != nil {
		return "No job alerts today", []JobAlert{}, nil
	}

	if err := json.Unmarshal(data, &alerts); err != nil {
		return "", []JobAlert{}, err
	}

	if len(alerts) == 0 {
		return "No job alerts today", []JobAlert{}, nil
	}

	var summaryLines []string
	summaryLines = append(summaryLines, fmt.Sprintf("*Total:* %d new jobs", len(alerts)))
	summaryLines = append(summaryLines, "")

	for _, alert := range alerts {
		line := fmt.Sprintf("• *%s* - %s", alert.Company, alert.Title)
		if alert.Location != "" {
			line += fmt.Sprintf(" (%s)", alert.Location)
		}
		summaryLines = append(summaryLines, line)
	}

	return strings.Join(summaryLines, "\n"), alerts, nil
}

func clearJobAlerts() error {
	return os.WriteFile(JobAlertsFile, []byte("[]"), 0644)
}

func handleJobCommand(config Config, flags *jobFlags) error {
	alert := JobAlert{
		Company:  flags.company,
		Title:    flags.title,
		Status:   flags.status,
		URL:      flags.url,
		Location: flags.location,
		Salary:   flags.salary,
		Resume:   flags.resume,
		Time:     time.Now().Format(time.RFC3339),
	}

	// Build rich notification message
	var msgLines []string
	msgLines = append(msgLines, fmt.Sprintf("*Company:* %s", alert.Company))
	msgLines = append(msgLines, fmt.Sprintf("*Role:* %s", alert.Title))
	msgLines = append(msgLines, fmt.Sprintf("*Status:* %s", alert.Status))

	if alert.Location != "" {
		msgLines = append(msgLines, fmt.Sprintf("*Location:* %s", alert.Location))
	}
	if alert.Salary != "" {
		msgLines = append(msgLines, fmt.Sprintf("*Salary:* %s", alert.Salary))
	}
	if alert.URL != "" {
		msgLines = append(msgLines, "")
		msgLines = append(msgLines, fmt.Sprintf("[View Job Posting](%s)", alert.URL))
	}
	if alert.Resume != "" {
		msgLines = append(msgLines, fmt.Sprintf("*Resume:* %s", alert.Resume))
	}

	message := strings.Join(msgLines, "\n")

	// Always send notification
	if err := sendTelegram(config, "💼 New Job Alert", message, AlertInfo); err != nil {
		return err
	}

	// Track for daily summary
	if err := trackJobAlert(alert); err != nil {
		return err
	}

	fmt.Printf("Job alert sent and tracked: %s - %s\n", alert.Company, alert.Title)
	return nil
}

type jobFlags struct {
	company  string
	title    string
	status   string
	url      string
	location string
	salary   string
	resume   string
}

func (j *jobFlags) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, use key=value")
	}
	switch parts[0] {
	case "company":
		j.company = parts[1]
	case "title":
		j.title = parts[1]
	case "status":
		j.status = parts[1]
	case "url":
		j.url = parts[1]
	case "location":
		j.location = parts[1]
	case "salary":
		j.salary = parts[1]
	case "resume":
		j.resume = parts[1]
	default:
		return fmt.Errorf("unknown key: %s", parts[0])
	}
	return nil
}

type rssFlags struct {
	count    int
	articles []RSSArticle
}

func handleRSSCommand(config Config, flags *rssFlags) error {
	// Build rich notification with article list
	var msgLines []string
	msgLines = append(msgLines, fmt.Sprintf("*%d* new articles fetched and saved to vault", flags.count))
	msgLines = append(msgLines, "")

	if len(flags.articles) > 0 {
		msgLines = append(msgLines, "*Recent articles:*")
		// Show up to 5 articles in notification
		maxShow := 5
		if len(flags.articles) < maxShow {
			maxShow = len(flags.articles)
		}
		for i := 0; i < maxShow; i++ {
			article := flags.articles[i]
			// Truncate title if too long
			title := article.Title
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			msgLines = append(msgLines, fmt.Sprintf("• [%s](%s)", title, article.URL))
		}
		if len(flags.articles) > maxShow {
			msgLines = append(msgLines, fmt.Sprintf("_... and %d more_", len(flags.articles)-maxShow))
		}
	}

	message := strings.Join(msgLines, "\n")

	if err := sendTelegram(config, "📰 RSS Update", message, AlertInfo); err != nil {
		return err
	}

	fmt.Printf("RSS notification sent: %d articles\n", flags.count)
	return nil
}

func handleDailySummaryCommand(config Config) error {
	summary, alerts, err := getDailyJobSummary()
	if err != nil {
		return err
	}

	if len(alerts) == 0 {
		summary = "No job alerts today 📭"
	}

	if err := sendTelegram(config, "📊 Daily Job Summary", summary, AlertSuccess); err != nil {
		return err
	}

	clearJobAlerts()
	fmt.Printf("Daily summary sent: %d jobs\n", len(alerts))
	return nil
}

func handleGenericCommand(config Config, title, message string, levelStr string) error {
	level := AlertInfo
	switch levelStr {
	case "success":
		level = AlertSuccess
	case "warning":
		level = AlertWarning
	case "error":
		level = AlertError
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

	// Job command flags
	jobCmd := flag.NewFlagSet("job", flag.ExitOnError)
	jobCompany := jobCmd.String("company", "", "Company name (required)")
	jobTitle := jobCmd.String("title", "", "Job title (required)")
	jobStatus := jobCmd.String("status", "New", "Status (e.g., Applied, Interview, Offer)")
	jobURL := jobCmd.String("url", "", "URL to job posting")
	jobLocation := jobCmd.String("location", "", "Job location")
	jobSalary := jobCmd.String("salary", "", "Salary range")
	jobResume := jobCmd.String("resume", "", "Resume filename used")

	// RSS command flags
	rssCmd := flag.NewFlagSet("rss", flag.ExitOnError)
	rssCount := rssCmd.Int("count", 0, "Number of articles")
	rssArticles := rssCmd.String("articles", "", "JSON array of articles [{title, url, source},...]")

	// Daily summary - no flags needed
	_ = flag.NewFlagSet("daily-summary", flag.ExitOnError)

	// Generic command flags
	genericCmd := flag.NewFlagSet("generic", flag.ExitOnError)
	genericTitle := genericCmd.String("title", "", "Notification title")
	genericMessage := genericCmd.String("message", "", "Notification message")
	genericLevel := genericCmd.String("level", "info", "Alert level: info, success, warning, error")

	if len(os.Args) < 2 {
		fmt.Println("Usage: notify <command> [flags]")
		fmt.Println("")
		fmt.Println("Commands:")
		fmt.Println("  job -company=<name> -title=<role> [-status=<status>] [-url=<url>] [-location=<loc>] [-salary=<$$>] [-resume=<file>]")
		fmt.Println("  rss -count=<n> [-articles='[{title,url,source},...]']")
		fmt.Println("  daily-summary")
		fmt.Println("  generic -title=<title> -message=<msg> [-level=<level>]")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  notify job -company=Google -title='SWE Intern' -url='https://...' -location='SF' -salary='$50/hr'")
		fmt.Println("  notify rss -count=15 -articles='[{title:AI News,url:https://...,source:HN}]'")
		os.Exit(1)
	}

	cmd := strings.ToLower(os.Args[1])
	var err error

	switch cmd {
	case "job":
		err = jobCmd.Parse(os.Args[2:])
		if err == nil {
			if *jobCompany == "" || *jobTitle == "" {
				fmt.Fprintln(os.Stderr, "Error: -company and -title are required")
				os.Exit(1)
			}
			flags := &jobFlags{
				company:  *jobCompany,
				title:    *jobTitle,
				status:   *jobStatus,
				url:      *jobURL,
				location: *jobLocation,
				salary:   *jobSalary,
				resume:   *jobResume,
			}
			err = handleJobCommand(config, flags)
		}
	case "rss":
		err = rssCmd.Parse(os.Args[2:])
		if err == nil {
			flags := &rssFlags{
				count: *rssCount,
			}
			if *rssArticles != "" {
				json.Unmarshal([]byte(*rssArticles), &flags.articles)
			}
			err = handleRSSCommand(config, flags)
		}
	case "daily-summary", "dailysummary":
		err = handleDailySummaryCommand(config)
	case "generic":
		err = genericCmd.Parse(os.Args[2:])
		if err == nil {
			if *genericTitle == "" || *genericMessage == "" {
				fmt.Fprintln(os.Stderr, "Error: -title and -message are required")
				os.Exit(1)
			}
			err = handleGenericCommand(config, *genericTitle, *genericMessage, *genericLevel)
		}
	default:
		err = fmt.Errorf("Unknown command: %s", cmd)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
