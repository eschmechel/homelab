package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type Feed struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type Config struct {
	Feeds       []Feed `yaml:"feeds"`
	VaultPath   string `yaml:"vault_path"`
	NewsFolder  string `yaml:"news_folder"`
	CacheDir    string `yaml:"cache_dir"`
	LLMEnabled  bool   `yaml:"llm_enabled"`
	LLMProvider string `yaml:"llm_provider"` // openai, anthropic, perplexity, ollama
	LLMModel    string `yaml:"llm_model"`
}

type Article struct {
	Title     string
	Link      string
	Summary   string
	Content   string
	Published string
	Source    string
	Hash      string
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
		articleHash := hashArticle(entry.Title, *entry.Link)

		if seenArticles.Hashes[articleHash] {
			continue
		}
		seenArticles.Hashes[articleHash] = true

		article := Article{
			Title:     entry.Title,
			Link:      *entry.Link,
			Summary:   entry.Description,
			Published: *entry.PublishedParsed,
			Source:    feed.Title,
			Hash:      articleHash,
		}

		if len(entry.Content) > 0 {
			article.Content = entry.Content
		} else {
			article.Content = entry.Description
		}

		articles = append(articles, article)
	}

	return articles, nil
}

func enhanceWithLLM(article *Article) error {
	if !config.LLMEnabled {
		return nil
	}

	switch config.LLMProvider {
	case "openai":
		return enhanceWithOpenAI(article)
	case "anthropic":
		return enhanceWithAnthropic(article)
	case "perplexity":
		return enhanceWithPerplexity(article)
	case "ollama":
		return enhanceWithOllama(article)
	default:
		return nil
	}
}

func enhanceWithOpenAI(article *Article) error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY not set")
	}

	prompt := fmt.Sprintf(`Analyze this article and provide:
1. A 2-sentence summary
2. Key entities (companies, people, technologies)
3. How this relates to AI/DevOps/Career development
4. 2-3 action items

Title: %s
Content: %s`, article.Title, article.Content[:2000])

	reqBody := map[string]interface{}{
		"model": config.LLMModel,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a knowledge assistant. Provide structured insights."},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 500,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", strings.NewReader(string(jsonData)))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Parse response and add to article
	// For now, just log
	fmt.Printf("LLM enhanced: %s\n", article.Title)
	return nil
}

func enhanceWithAnthropic(article *Article) error {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	fmt.Printf("Anthropic enhancement for: %s\n", article.Title)
	return nil
}

func enhanceWithPerplexity(article *Article) error {
	apiKey := os.Getenv("PERPLEXITY_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("PERPLEXITY_API_KEY not set")
	}

	fmt.Printf("Perplexity research for: %s\n", article.Title)
	return nil
}

func enhanceWithOllama(article *Article) error {
	fmt.Printf("Ollama enhancement for: %s\n", article.Title)
	return nil
}

func writeArticleToVault(article Article) error {
	safeTitle := strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == ' ' {
			return r
		}
		return -1
	}, article.Title)

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

	publishedStr := article.Published
	if publishedStr == "" {
		publishedStr = time.Now().Format("2006-01-02T15:04:05Z07:00")
	}

	content := fmt.Sprintf(`---
date: %s
tags: [rss, %s]
---

# %s

**Source:** %s
**Published:** %s
**Link:** [Read Original](%s)

## Summary

%s

`,
		dateStr,
		strings.ToLower(strings.ReplaceAll(article.Source, " ", "-")),
		article.Title,
		article.Source,
		publishedStr,
		article.Link,
		article.Summary,
	)

	return os.WriteFile(filepath, []byte(content), 0644)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: rss-fetcher <config.json>")
		os.Exit(1)
	}

	configPath := os.Args[1]

	// Set defaults
	config = Config{
		VaultPath:   "/mnt/GOONDRIVE/Obsidian Notes/Personal Knowledge System",
		NewsFolder:  "40-Knowledge/News",
		CacheDir:    "/tmp/rss_cache",
		LLMEnabled:  false,
		LLMProvider: "ollama",
		LLMModel:    "llama3",
	}

	if err := loadConfig(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := loadSeenArticles(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load cache: %v\n", err)
	}

	// Create cache dir
	os.MkdirAll(config.CacheDir, 0755)

	var allArticles []Article

	for _, feed := range config.Feeds {
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

	// Write articles to vault
	for _, article := range allArticles {
		if err := writeArticleToVault(article); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing article: %v\n", err)
			continue
		}

		// Optionally enhance with LLM
		if config.LLMEnabled {
			enhanceWithLLM(&article)
		}
	}

	fmt.Printf("\nTotal: %d new articles saved to vault\n", len(allArticles))
}
