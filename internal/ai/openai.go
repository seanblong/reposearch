package ai

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type OpenAIClient struct {
	config *ClientConfig
	http   *http.Client
}

func NewOpenAIClient(config *ClientConfig) *OpenAIClient {
	// Set default models if not provided
	if config.EmbedModel == "" {
		config.EmbedModel = "text-embedding-3-small"
	}
	if config.SummaryModel == "" {
		config.SummaryModel = "gpt-4o-mini"
	}
	if config.Dim == 0 {
		// Set default dimensions based on the embedding model
		switch config.EmbedModel {
		case "text-embedding-3-small":
			config.Dim = 1536
		case "text-embedding-3-large":
			config.Dim = 3072
		case "text-embedding-ada-002":
			config.Dim = 1536
		default:
			// Default to text-embedding-3-small dimensions
			config.Dim = 1536
		}
	}

	// Create HTTP client with optional TLS skip verification
	transport := &http.Transport{}

	// Check for environment variable to skip TLS verification (for corporate proxies, etc.)
	if skipTLS, _ := strconv.ParseBool(os.Getenv("REPOSEARCH_SKIP_TLS_VERIFY")); skipTLS {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	httpClient := &http.Client{
		Timeout:   20 * time.Second,
		Transport: transport,
	}

	return &OpenAIClient{
		config: config,
		http:   httpClient,
	}
}

// Embed implements the embedding functionality
func (c *OpenAIClient) Embed(text string) ([]float32, error) {
	if c.config.APIKey == "" {
		return nil, errors.New("PROVIDER_API_KEY unset")
	}

	payload := map[string]string{
		"input": text,
		"model": c.config.EmbedModel,
	}

	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
		"https://api.openai.com/v1/embeddings", bytes.NewReader(b))

	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("openai embedding non-200")
	}

	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, errors.New("no embedding")
	}
	return out.Data[0].Embedding, nil
}

// Summarize implements the summarization functionality
func (c *OpenAIClient) Summarize(ctx context.Context, filePath, language, content string) (string, error) {
	if c.config.APIKey == "" {
		return "", errors.New("PROVIDER_API_KEY unset")
	}

	// Keep request small; the model only needs a taste
	const maxInput = 8000
	if len(content) > maxInput {
		content = content[:maxInput]
	}

	sys := "You are a concise code summarizer. Write at most 240 characters, 1–2 sentences, no code blocks, no backticks. Mention the file's purpose and notable actions. Prefer verbs. If the text is configuration, say what it configures."
	user := "Path: " + filePath + "\nLanguage: " + language + "\n---\n" + content

	payload := map[string]any{
		"model": c.config.SummaryModel,
		"messages": []map[string]string{
			{"role": "system", "content": sys},
			{"role": "user", "content": user},
		},
		"temperature": 0.2,
		"max_tokens":  120,
	}

	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(payload)

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.openai.com/v1/chat/completions", &buf)
	if err != nil {
		return "", err
	}

	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e struct{ Error struct{ Message string } }
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error.Message != "" {
			return "", errors.New(e.Error.Message)
		}
		return "", errors.New(resp.Status)
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("no choices")
	}

	s := strings.TrimSpace(out.Choices[0].Message.Content)
	s = strings.ReplaceAll(s, "\n", " ")
	return s, nil
}

func (c *OpenAIClient) Dim() int {
	return c.config.Dim
}

// setHeaders sets common headers for OpenAI requests
func (c *OpenAIClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	if strings.HasPrefix(c.config.APIKey, "sk-proj-") && c.config.ProjectID != "" {
		req.Header.Set("OpenAI-Project", c.config.ProjectID)
	}
}
