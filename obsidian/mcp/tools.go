package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func matchRegex(pattern, content string) bool {
	re := regexp.MustCompile(pattern)
	return re.FindString(content) != ""
}

func findDateMatches(pattern, content, startDate, endDate string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(content, -1)

	var results []string
	for _, match := range matches {
		if len(match) > 1 {
			date := match[1]
			if startDate != "" && date >= startDate {
				if endDate == "" || date <= endDate {
					results = append(results, date)
				}
			}
		}
	}
	return results
}

func getTelegramConfig() (string, string, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if token == "" {
		return "", "", fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}
	if chatID == "" {
		return "", "", fmt.Errorf("TELEGRAM_CHAT_ID not set")
	}

	return token, chatID, nil
}

func sendTelegramMessage(title, message, level string) error {
	token, chatID, err := getTelegramConfig()
	if err != nil {
		return err
	}

	emoji := "ℹ️"
	switch level {
	case "success":
		emoji = "✅"
	case "warning":
		emoji = "⚠️"
	case "error":
		emoji = "❌"
	}

	text := fmt.Sprintf("%s *%s*\n\n%s", emoji, title, message)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)

	msg := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	jsonData, _ := json.Marshal(msg)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

func sendTelegramNotification(title, message, level string) (string, error) {
	if err := sendTelegramMessage(title, message, level); err != nil {
		return "", err
	}
	return fmt.Sprintf("Notification sent: %s", title), nil
}

func sendJobAlert(company, status string, notifyIndividual bool) (string, error) {
	if notifyIndividual {
		if err := sendTelegramMessage("Job Alert", fmt.Sprintf("%s: %s", company, status), "info"); err != nil {
			return "", err
		}
	}

	// Track for daily summary - store in a temp file
	summaryFile := "/tmp/job_alerts_today.json"
	var alerts []map[string]string

	if data, err := os.ReadFile(summaryFile); err == nil {
		json.Unmarshal(data, &alerts)
	}

	alerts = append(alerts, map[string]string{
		"company": company,
		"status":  status,
		"time":    time.Now().Format(time.RFC3339),
	})

	if data, err := json.Marshal(alerts); err == nil {
		os.WriteFile(summaryFile, data, 0644)
	}

	if notifyIndividual {
		return fmt.Sprintf("Job alert sent for %s: %s", company, status), nil
	}
	return fmt.Sprintf("Job alert tracked (notify=no): %s - %s", company, status), nil
}

func sendRSSNotification(count string, notify bool) (string, error) {
	if notify {
		if err := sendTelegramMessage("RSS Update", fmt.Sprintf("%s new articles fetched and saved to vault", count), "info"); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("RSS notification: %s articles (notify=%v)", count, notify), nil
}

func GetDailyJobSummary() (string, error) {
	summaryFile := "/tmp/job_alerts_today.json"
	var alerts []map[string]string

	data, err := os.ReadFile(summaryFile)
	if err != nil {
		return "No job alerts today", nil
	}

	if err := json.Unmarshal(data, &alerts); err != nil {
		return "", err
	}

	if len(alerts) == 0 {
		return "No job alerts today", nil
	}

	summary := fmt.Sprintf("📊 Daily Job Summary\n\nTotal: %d alerts\n\n", len(alerts))
	for _, alert := range alerts {
		summary += fmt.Sprintf("• %s: %s\n", alert["company"], alert["status"])
	}

	// Clear the file
	os.WriteFile(summaryFile, []byte("[]"), 0644)

	return summary, nil
}

func handleInitialize(id interface{}) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"serverInfo": map[string]any{
				"name":    "obsidian-filesystem",
				"version": "0.1.0",
			},
		},
	}
}

func handleToolsList(id interface{}) JSONRPCResponse {
	tools := []map[string]any{
		{
			"name":        "list_directory",
			"description": "List files and directories in a folder",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string", "description": "Relative path to directory"},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "read_file",
			"description": "Read contents of a file",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string", "description": "Relative path to file"},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "write_file",
			"description": "Write content to a file",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    map[string]any{"type": "string", "description": "Relative path to file"},
					"content": map[string]any{"type": "string", "description": "Content to write"},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			"name":        "delete_file",
			"description": "Delete a file",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string", "description": "Relative path to file"},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "create_directory",
			"description": "Create a new directory",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string", "description": "Relative path to directory"},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "search_files",
			"description": "Search for files by name pattern",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{"type": "string", "description": "Glob pattern"},
					"path":    map[string]any{"type": "string", "description": "Directory to search"},
				},
				"required": []string{"pattern"},
			},
		},
		{
			"name":        "search_content",
			"description": "Search for text within files",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Text to search for"},
					"path":  map[string]any{"type": "string", "description": "Directory to search"},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "move_file",
			"description": "Move or rename a file",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source": map[string]any{"type": "string", "description": "Source path"},
					"dest":   map[string]any{"type": "string", "description": "Destination path"},
				},
				"required": []string{"source", "dest"},
			},
		},
		{
			"name":        "get_file_info",
			"description": "Get file metadata",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string", "description": "Relative path to file"},
				},
				"required": []string{"path"},
			},
		},
		{
			"name":        "search_by_tag",
			"description": "Search for notes by YAML frontmatter tags using ripgrep",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tag":  map[string]any{"type": "string", "description": "Tag to search for"},
					"path": map[string]any{"type": "string", "description": "Directory to search (default: vault root)"},
				},
				"required": []string{"tag"},
			},
		},
		{
			"name":        "search_by_date",
			"description": "Search for notes by date range in YAML frontmatter",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"start_date": map[string]any{"type": "string", "description": "Start date (YYYY-MM-DD)"},
					"end_date":   map[string]any{"type": "string", "description": "End date (YYYY-MM-DD)"},
					"path":       map[string]any{"type": "string", "description": "Directory to search"},
				},
				"required": []string{"start_date"},
			},
		},
		{
			"name":        "send_notification",
			"description": "Send a notification via Telegram",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":   map[string]any{"type": "string", "description": "Notification title"},
					"message": map[string]any{"type": "string", "description": "Notification message"},
					"level":   map[string]any{"type": "string", "description": "Alert level: info, success, warning, error"},
				},
				"required": []string{"title", "message"},
			},
		},
		{
			"name":        "send_job_alert",
			"description": "Send a job application alert notification",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"company": map[string]any{"type": "string", "description": "Company name"},
					"status":  map[string]any{"type": "string", "description": "Job status (e.g., Applied, Interview, Offer)"},
					"notify":  map[string]any{"type": "string", "description": "Send individual notification: yes or no"},
				},
				"required": []string{"company", "status"},
			},
		},
		{
			"name":        "send_rss_notification",
			"description": "Send RSS fetch summary notification",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"count":  map[string]any{"type": "string", "description": "Number of new articles"},
					"notify": map[string]any{"type": "string", "description": "Send notification: yes or no"},
				},
				"required": []string{"count"},
			},
		},
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"tools": tools,
		},
	}
}

func handleToolsCall(id interface{}, params map[string]any) JSONRPCResponse {
	arguments, _ := params["arguments"].(map[string]any)
	if arguments == nil {
		arguments = params
	}

	toolName, _ := params["name"].(string)

	switch toolName {
	case "list_directory":
		path, _ := arguments["path"].(string)
		fullPath := resolvePath(path)
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		var files []string
		for _, e := range entries {
			files = append(files, e.Name())
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("%v", files)},
				},
			},
		}

	case "read_file":
		path, _ := arguments["path"].(string)
		fullPath := resolvePath(path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": string(data)},
				},
			},
		}

	case "write_file":
		path, _ := arguments["path"].(string)
		content, _ := arguments["content"].(string)
		fullPath := resolvePath(path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("File written: %s", path)},
				},
			},
		}

	case "delete_file":
		path, _ := arguments["path"].(string)
		fullPath := resolvePath(path)
		if err := os.Remove(fullPath); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("File deleted: %s", path)},
				},
			},
		}

	case "create_directory":
		path, _ := arguments["path"].(string)
		fullPath := resolvePath(path)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("Directory created: %s", path)},
				},
			},
		}

	case "search_files":
		pattern, _ := arguments["pattern"].(string)
		searchPath, _ := arguments["path"].(string)
		fullPath := vaultPath
		if searchPath != "" {
			fullPath = resolvePath(searchPath)
		}
		var results []string
		filepath.Walk(fullPath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			name := filepath.Base(p)
			matched, _ := filepath.Match(pattern, name)
			if matched {
				rel, _ := filepath.Rel(vaultPath, p)
				results = append(results, rel)
			}
			return nil
		})
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": strings.Join(results, "\n")},
				},
			},
		}

	case "search_content":
		query, _ := arguments["query"].(string)
		searchPath, _ := arguments["path"].(string)
		fullPath := vaultPath
		if searchPath != "" {
			fullPath = resolvePath(searchPath)
		}
		var results []string
		filepath.Walk(fullPath, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			if strings.Contains(string(data), query) {
				rel, _ := filepath.Rel(vaultPath, p)
				results = append(results, rel)
			}
			return nil
		})
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": strings.Join(results, "\n")},
				},
			},
		}

	case "move_file":
		source, _ := arguments["source"].(string)
		dest, _ := arguments["dest"].(string)
		srcPath := resolvePath(source)
		dstPath := resolvePath(dest)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		if err := os.Rename(srcPath, dstPath); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("File moved: %s -> %s", source, dest)},
				},
			},
		}

	case "get_file_info":
		path, _ := arguments["path"].(string)
		fullPath := resolvePath(path)
		info, err := os.Stat(fullPath)
		if err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: fmt.Sprintf("Error: %v", err)},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": fmt.Sprintf("Name: %s\nSize: %d bytes\nModified: %s\nIs Directory: %v", info.Name(), info.Size(), info.ModTime().Format(time.RFC3339), info.IsDir())},
				},
			},
		}

	case "search_by_tag":
		tag, _ := arguments["tag"].(string)
		searchPath, _ := arguments["path"].(string)
		fullPath := vaultPath
		if searchPath != "" {
			fullPath = resolvePath(searchPath)
		}

		var results []string
		filepath.Walk(fullPath, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			content := string(data)
			tagPattern := fmt.Sprintf(`tags:.*%s`, tag)
			if strings.Contains(content, tag) || matchRegex(tagPattern, content) {
				rel, _ := filepath.Rel(vaultPath, p)
				results = append(results, rel)
			}
			return nil
		})
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": strings.Join(results, "\n")},
				},
			},
		}

	case "search_by_date":
		startDate, _ := arguments["start_date"].(string)
		endDate, _ := arguments["end_date"].(string)
		searchPath, _ := arguments["path"].(string)
		fullPath := vaultPath
		if searchPath != "" {
			fullPath = resolvePath(searchPath)
		}

		var results []string
		filepath.Walk(fullPath, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			content := string(data)
			datePattern := `date:\s*(\d{4}-\d{2}-\d{2})`
			matches := findDateMatches(datePattern, content, startDate, endDate)
			if len(matches) > 0 {
				rel, _ := filepath.Rel(vaultPath, p)
				results = append(results, rel)
			}
			return nil
		})
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": strings.Join(results, "\n")},
				},
			},
		}

	case "send_notification":
		title, _ := arguments["title"].(string)
		message, _ := arguments["message"].(string)
		level, _ := arguments["level"].(string)

		result, err := sendTelegramNotification(title, message, level)
		if err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: err.Error()},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": result},
				},
			},
		}

	case "send_job_alert":
		company, _ := arguments["company"].(string)
		status, _ := arguments["status"].(string)
		notify, _ := arguments["notify"].(string)

		result, err := sendJobAlert(company, status, notify == "yes")
		if err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: err.Error()},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": result},
				},
			},
		}

	case "send_rss_notification":
		count, _ := arguments["count"].(string)
		notify, _ := arguments["notify"].(string)

		result, err := sendRSSNotification(count, notify == "yes")
		if err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error:   &JSONError{Code: -32602, Message: err.Error()},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Result: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": result},
				},
			},
		}
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &JSONError{Code: -32602, Message: "Unknown tool"},
	}
}
