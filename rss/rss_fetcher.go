package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type Feed struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

type Config struct {
	Feeds       []Feed `json:"feeds"`
	VaultPath   string `json:"vault_path"`
	NewsFolder  string `json:"news_folder"`
	CacheDir    string `json:"cache_dir"`
	LLMEnabled  bool   `json:"llm_enabled"`
	LLMProvider string `json:"llm_provider"` // perplexity (default), minimax
	LLMModel    string `json:"llm_model"`
}

type Article struct {
	Title     string `json:"title"`
	Link      string `json:"link"`
	Summary   string `json:"summary"`
	Content   string `json:"content"`
	Published string `json:"published"`
	Source    string `json:"source"`
	Hash      string `json:"hash"`
}

type EnhancedArticle struct {
	Article
	LLMSummary      string   `json:"llm_summary"`
	KeyInsights     []string `json:"key_insights"`
	RelatedConcepts []string `json:"related_concepts"`
	ActionItems     []string `json:"action_items"`
	Citations       []string `json:"citations"`
}

type SeenArticles struct {
	Hashes map[string]bool `json:"hashes"`
}

var (
	config       Config
	seenArticles SeenArticles
)

func loadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &config)
}

func loadSeenArticles() error {
	cacheFile := filepath.Join(config.CacheDir, "seen.json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			seenArticles.Hashes = make(map[string]bool)
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &seenArticles)
}

func saveSeenArticles() error {
	cacheFile := filepath.Join(config.CacheDir, "seen.json")
	dir := filepath.Dir(cacheFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(seenArticles)
	if err != nil {
		return err
	}
	return os.WriteFile(cacheFile, data, 0644)
}

func hashArticle(title, link string) string {
	data := title + link
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}

func fetchFeed(url string) ([]Article, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	parser := gofeed.NewParser()
	feed, err := parser.ParseString(string(body))
	if err != nil {
		return nil, err
	}

	var articles []Article
	for _, entry := range feed.Items {
		link := entry.Link // gofeed returns string directly
		articleHash := hashArticle(entry.Title, link)

		if seenArticles.Hashes[articleHash] {
			continue
		}
		seenArticles.Hashes[articleHash] = true

		published := ""
		if entry.PublishedParsed != nil {
			published = entry.PublishedParsed.Format(time.RFC3339)
		}

		content := entry.Description
		if len(entry.Content) > 0 {
			content = entry.Content
		}

		article := Article{
			Title:     entry.Title,
			Link:      link,
			Summary:   entry.Description,
			Content:   content,
			Published: published,
			Source:    feed.Title,
			Hash:      articleHash,
		}

		articles = append(articles, article)
	}

	return articles, nil
}

func enhanceWithLLM(article *Article) *EnhancedArticle {
	if !config.LLMEnabled {
		return &EnhancedArticle{Article: *article}
	}

	// Use Perplexity by default, MiniMax as backup
	provider := config.LLMProvider
	if provider == "" {
		provider = "perplexity"
	}

	apiKey := os.Getenv("PERPLEXITY_API_KEY")
	if provider == "minimax" {
		apiKey = os.Getenv("MINIMAX_API_KEY")
	}

	if apiKey == "" {
		fmt.Printf("Warning: No LLM API key set for %s\n", provider)
		return &EnhancedArticle{Article: *article}
	}

	// Build the prompt
	prompt := fmt.Sprintf(`Analyze this article and provide a structured response with these exact JSON keys:
{"summary": "2-3 sentence summary", "key_insights": ["insight1", "insight2"], "related_concepts": ["concept1"], "action_items": ["action1"]}

Article Title: %s
Article Content (first 3000 chars): %s

Respond ONLY with valid JSON, no other text.`, article.Title, article.Content[:min(len(article.Content), 3000)])

	var respBody []byte

	if provider == "perplexity" {
		model := config.LLMModel
		if model == "" {
			model = "llama-3.1-sonar-small-128k-online"
		}

		reqBody := map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "system", "content": "You are a knowledge analysis assistant. Provide structured, actionable insights in JSON format."},
				{"role": "user", "content": prompt},
			},
			"max_tokens": 1000,
		}
		jsonData, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "https://api.perplexity.ai/chat/completions", strings.NewReader(string(jsonData)))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error calling Perplexity: %v\n", err)
			return &EnhancedArticle{Article: *article}
		}
		defer resp.Body.Close()
		respBody, _ = io.ReadAll(resp.Body)

		// Parse Perplexity response
		var perplexityResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		json.Unmarshal(respBody, &perplexityResp)
		if len(perplexityResp.Choices) > 0 {
			content := perplexityResp.Choices[0].Message.Content
			// Extract JSON from response
			start := strings.Index(content, "{")
			end := strings.LastIndex(content, "}")
			if start >= 0 && end >= start {
				respBody = []byte(content[start : end+1])
			}
		}
	} else {
		// MiniMax implementation
		fmt.Printf("MiniMax enhancement not fully implemented, skipping LLM\n")
		return &EnhancedArticle{Article: *article}
	}

	// Parse the LLM response
	var llmResp struct {
		Summary         string   `json:"summary"`
		KeyInsights     []string `json:"key_insights"`
		RelatedConcepts []string `json:"related_concepts"`
		ActionItems     []string `json:"action_items"`
	}

	if err := json.Unmarshal(respBody, &llmResp); err != nil {
		fmt.Printf("Warning: Failed to parse LLM response: %v\n", err)
		return &EnhancedArticle{Article: *article}
	}

	return &EnhancedArticle{
		Article:         *article,
		LLMSummary:      llmResp.Summary,
		KeyInsights:     llmResp.KeyInsights,
		RelatedConcepts: llmResp.RelatedConcepts,
		ActionItems:     llmResp.ActionItems,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func writeArticleToVault(enhanced EnhancedArticle) error {
	safeTitle := strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' {
			return r
		}
		return -1
	}, enhanced.Title)

	if len(safeTitle) > 50 {
		safeTitle = safeTitle[:50]
	}

	dateStr := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.md", dateStr, strings.ReplaceAll(safeTitle, " ", "-"))

	newsPath := filepath.Join(config.VaultPath, config.NewsFolder)
	if err := os.MkdirAll(newsPath, 0755); err != nil {
		return err
	}

	filepath := filepath.Join(newsPath, filename)

	publishedStr := enhanced.Published
	if publishedStr == "" {
		publishedStr = time.Now().Format("2006-01-02T15:04:05Z07:00")
	}

	// Build markdown content
	var lines []string
	lines = append(lines, "---")
	lines = append(lines, fmt.Sprintf("date: %s", dateStr))
	lines = append(lines, fmt.Sprintf("tags: [rss, %s]", strings.ToLower(strings.ReplaceAll(enhanced.Source, " ", "-"))))
	lines = append(lines, "---")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("# %s", enhanced.Title))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("**Source:** %s", enhanced.Source))
	lines = append(lines, fmt.Sprintf("**Published:** %s", publishedStr))
	lines = append(lines, fmt.Sprintf("**Link:** [Read Original](%s)", enhanced.Link))
	lines = append(lines, "")

	if enhanced.LLMSummary != "" {
		lines = append(lines, "## Summary")
		lines = append(lines, "")
		lines = append(lines, enhanced.LLMSummary)
		lines = append(lines, "")
	}

	if len(enhanced.KeyInsights) > 0 {
		lines = append(lines, "## Key Insights")
		lines = append(lines, "")
		for _, insight := range enhanced.KeyInsights {
			lines = append(lines, fmt.Sprintf("- %s", insight))
		}
		lines = append(lines, "")
	}

	if len(enhanced.RelatedConcepts) > 0 {
		lines = append(lines, "## Related Concepts")
		lines = append(lines, "")
		for _, concept := range enhanced.RelatedConcepts {
			lines = append(lines, fmt.Sprintf("- %s", concept))
		}
		lines = append(lines, "")
	}

	if len(enhanced.ActionItems) > 0 {
		lines = append(lines, "## Action Items")
		lines = append(lines, "")
		for _, item := range enhanced.ActionItems {
			lines = append(lines, fmt.Sprintf("- [ ] %s", item))
		}
		lines = append(lines, "")
	}

	return os.WriteFile(filepath, []byte(strings.Join(lines, "\n")), 0644)
}

func sendNotification(count int, articles []Article) error {
	// Build article JSON for notification
	var articleJSON string
	if len(articles) > 0 {
		var articleMaps []map[string]string
		for _, a := range articles {
			if len(articleMaps) < 5 { // Only include first 5 in notification
				articleMaps = append(articleMaps, map[string]string{
					"title":  a.Title,
					"url":    a.Link,
					"source": a.Source,
				})
			}
		}
		jsonData, _ := json.Marshal(articleMaps)
		articleJSON = string(jsonData)
	}

	// Call the notification CLI
	notifyPath := os.Getenv("NOTIFY_BIN")
	if notifyPath == "" {
		notifyPath = "notify"
	}

	args := []string{"rss", "-count", fmt.Sprintf("%d", count)}
	if articleJSON != "" {
		args = append(args, "-articles", articleJSON)
	}

	cmd := exec.Command(notifyPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("notification failed: %v, output: %s", err, string(output))
	}

	fmt.Printf("Notification sent: %s\n", string(output))
	return nil
}

func main() {
	configPath := flag.String("config", "config.json", "Path to config file")
	flag.Parse()

	// Set defaults
	config = Config{
		VaultPath:   "/mnt/GOONDRIVE/Obsidian Notes/Personal Knowledge System",
		NewsFolder:  "40-Knowledge/News",
		CacheDir:    "/tmp/rss_cache",
		LLMEnabled:  true,
		LLMProvider: "perplexity", // Default to Perplexity
		LLMModel:    "llama-3.1-sonar-small-128k-online",
	}

	if err := loadConfig(*configPath); err != nil {
		// Try to load from args
		if len(os.Args) > 1 {
			if err := loadConfig(os.Args[1]); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		}
	}

	if err := loadSeenArticles(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load cache: %v\n", err)
	}

	// Create cache dir
	os.MkdirAll(config.CacheDir, 0755)

	var allArticles []Article

	for _, feed := range config.Feeds {
		if !feed.Enabled {
			continue
		}
		fmt.Printf("Fetching: %s\n", feed.Name)
		articles, err := fetchFeed(feed.URL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching %s: %v\n", feed.Name, err)
			continue
		}
		allArticles = append(allArticles, articles...)
		fmt.Printf("Found %d new articles from %s\n", len(articles), feed.Name)
	}

	// Save seen articles
	saveSeenArticles()

	// Process articles
	var processedArticles []Article
	for _, article := range allArticles {
		// Enhance with LLM (Perplexity by default)
		enhanced := enhanceWithLLM(&article)

		if err := writeArticleToVault(*enhanced); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing article: %v\n", err)
			continue
		}
		processedArticles = append(processedArticles, article)
		fmt.Printf("Saved: %s\n", article.Title)
	}

	fmt.Printf("\nTotal: %d new articles saved to vault\n", len(allArticles))

	// Send notification
	if len(processedArticles) > 0 {
		if err := sendNotification(len(processedArticles), processedArticles); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to send notification: %v\n", err)
		}
	}
}
