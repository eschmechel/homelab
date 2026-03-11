package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type LLMProvider interface {
	Enhance(article *Article) (*EnhancedArticle, error)
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

type EnhancedArticle struct {
	Article         Article
	LLMSummary      string
	KeyInsights     []string
	RelatedConcepts []string
	ActionItems     []string
	Citations       []string
}

type PerplexityConfig struct {
	APIKey    string
	Model     string
	MaxTokens int
}

type PerplexityRequest struct {
	Model           string    `json:"model"`
	Messages        []Message `json:"messages"`
	MaxTokens       int       `json:"max_tokens"`
	ReturnCitations bool      `json:"return_citations"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type PerplexityResponse struct {
	Choices   []Choice `json:"choices"`
	Usage     Usage    `json:"usage"`
	Citations []string `json:"citations"`
}

type Choice struct {
	Message Message `json:"message"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type PerplexityProvider struct {
	config PerplexityConfig
	client *http.Client
}

func NewPerplexityProvider(apiKey string, model string) *PerplexityProvider {
	return &PerplexityProvider{
		config: PerplexityConfig{
			APIKey:    apiKey,
			Model:     model,
			MaxTokens: 1000,
		},
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *PerplexityProvider) Enhance(article *Article) (*EnhancedArticle, error) {
	prompt := fmt.Sprintf(`Analyze this article and provide a structured response with the following sections:
1. SUMMARY: A 2-3 sentence summary of the key points
2. KEY_INSIGHTS: 3-5 bullet points of the most important insights
3. RELATED_CONCEPTS: 3-5 related topics or concepts this connects to
4. ACTION_ITEMS: 2-3 specific actions this reader should consider

Article Title: %s
Article Content: %s

Provide your response in JSON format with keys: summary, key_insights, related_concepts, action_items`,
		article.Title, article.Content[:3000])

	reqBody := PerplexityRequest{
		Model: p.config.Model,
		Messages: []Message{
			{Role: "system", Content: "You are a knowledge analysis assistant. Provide structured, actionable insights."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:       p.config.MaxTokens,
		ReturnCitations: true,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.perplexity.ai/chat/completions", bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Perplexity API error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Perplexity API error (status %d): %s", resp.StatusCode, string(body))
	}

	var perplexityResp PerplexityResponse
	if err := json.Unmarshal(body, &perplexityResp); err != nil {
		return nil, fmt.Errorf("Failed to parse Perplexity response: %w", err)
	}

	if len(perplexityResp.Choices) == 0 {
		return nil, fmt.Errorf("No response from Perplexity")
	}

	// Parse the JSON response from Perplexity
	content := perplexityResp.Choices[0].Message.Content
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		// If not JSON, create basic enhanced article
		return &EnhancedArticle{
			Article:     *article,
			LLMSummary:  content,
			KeyInsights: []string{},
			ActionItems: []string{},
			Citations:   perplexityResp.Citations,
		}, nil
	}

	enhanced := &EnhancedArticle{
		Article:   *article,
		Citations: perplexityResp.Citations,
	}

	if s, ok := parsed["summary"].(string); ok {
		enhanced.LLMSummary = s
	}

	if insights, ok := parsed["key_insights"].([]interface{}); ok {
		for _, i := range insights {
			if s, ok := i.(string); ok {
				enhanced.KeyInsights = append(enhanced.KeyInsights, s)
			}
		}
	}

	if concepts, ok := parsed["related_concepts"].([]interface{}); ok {
		for _, c := range concepts {
			if s, ok := c.(string); ok {
				enhanced.RelatedConcepts = append(enhanced.RelatedConcepts, s)
			}
		}
	}

	if actions, ok := parsed["action_items"].([]interface{}); ok {
		for _, a := range actions {
			if s, ok := a.(string); ok {
				enhanced.ActionItems = append(enhanced.ActionItems, s)
			}
		}
	}

	return enhanced, nil
}

type MiniMaxProvider struct {
	APIKey string
	Model  string
	client *http.Client
}

func NewMiniMaxProvider(apiKey string, model string) *MiniMaxProvider {
	return &MiniMaxProvider{
		APIKey: apiKey,
		Model:  model,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (m *MiniMaxProvider) Enhance(article *Article) (*EnhancedArticle, error) {
	// MiniMax implementation would go here
	// For now, return basic enhanced article
	return &EnhancedArticle{
		Article:    *article,
		LLMSummary: article.Summary[:min(len(article.Summary), 500)],
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getLLMProvider() (LLMProvider, error) {
	perplexityKey := os.Getenv("PERPLEXITY_API_KEY")
	minimaxKey := os.Getenv("MINIMAX_API_KEY")

	// Prefer Perplexity if available
	if perplexityKey != "" {
		model := os.Getenv("PERPLEXITY_MODEL")
		if model == "" {
			model = "llama-3.1-sonar-small-128k-online"
		}
		return NewPerplexityProvider(perplexityKey, model), nil
	}

	if minimaxKey != "" {
		model := os.Getenv("MINIMAX_MODEL")
		if model == "" {
			model = "MiniMax-M2.5"
		}
		return NewMiniMaxProvider(minimaxKey, model), nil
	}

	return nil, fmt.Errorf("No LLM provider configured. Set PERPLEXITY_API_KEY or MINIMAX_API_KEY")
}

func formatEnhancedArticle(enhanced *EnhancedArticle) string {
	var lines []string

	lines = append(lines, "---")
	lines = append(lines, fmt.Sprintf("date: %s", time.Now().Format("2006-01-02")))
	lines = append(lines, fmt.Sprintf("tags: [rss, %s]", strings.ToLower(strings.ReplaceAll(enhanced.Article.Source, " ", "-"))))
	lines = append(lines, "---")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("# %s", enhanced.Article.Title))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("**Source:** %s", enhanced.Article.Source))
	lines = append(lines, fmt.Sprintf("**Published:** %s", enhanced.Article.Published))
	lines = append(lines, fmt.Sprintf("**Link:** [Read Original](%s)", enhanced.Article.Link))
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

	if len(enhanced.Citations) > 0 {
		lines = append(lines, "## Citations")
		lines = append(lines, "")
		for _, citation := range enhanced.Citations {
			lines = append(lines, fmt.Sprintf("- %s", citation))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
