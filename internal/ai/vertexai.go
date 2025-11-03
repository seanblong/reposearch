package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

type VertexAIClient struct {
	config *ClientConfig
	client *genai.Client
}

// NewVertexAIClient creates a new client for the Google Gemini API.
func NewVertexAIClient(ctx context.Context, config *ClientConfig) (*VertexAIClient, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	// Defaults for Gemini API
	if config.EmbedModel == "" {
		config.EmbedModel = "text-embedding-005"
	}
	if config.SummaryModel == "" {
		config.SummaryModel = "gemini-2.0-flash"
	}
	if config.Dim == 0 {
		config.Dim = 768
	}
	if config.Location == "" && strings.TrimSpace(config.APIKey) == "" {
		config.Location = "us-central1"
	}

	var client *genai.Client
	var err error
	cc := genai.ClientConfig{
		Backend: genai.BackendVertexAI,
	}

	if strings.TrimSpace(config.APIKey) != "" {
		cc.APIKey = config.APIKey
	}
	if strings.TrimSpace(config.ProjectID) != "" {
		cc.Project = config.ProjectID
	}
	if strings.TrimSpace(config.Location) != "" {
		cc.Location = config.Location
	}

	client, err = genai.NewClient(ctx, &cc)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &VertexAIClient{
		config: config,
		client: client,
	}, nil
}

// Close the client when done
func (c *VertexAIClient) Close() error {
	// return c.client.Close()
	return nil
}

// Embed implements the embedding functionality using the Gemini API
func (c *VertexAIClient) Embed(text string) ([]float32, error) {
	ctx := context.Background()
	cfg := genai.EmbedContentConfig{
		TaskType: "RETRIEVAL_DOCUMENT",
	}

	res, err := c.client.Models.EmbedContent(ctx, c.config.EmbedModel, genai.Text(text), &cfg)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	if res == nil || res.Embeddings == nil || len(res.Embeddings) == 0 {
		return nil, errors.New("no embedding returned")
	}

	return res.Embeddings[0].Values, nil
}

// Summarize implements the summarization functionality using the Gemini API
func (c *VertexAIClient) Summarize(ctx context.Context, filePath, language, content string) (string, error) {
	// Keep request small; the model only needs a taste
	const maxInput = 8000
	if len(content) > maxInput {
		content = content[:maxInput]
	}

	prompt := genai.Text("You are a concise code summarizer. Write at most 240 characters, 1–2 sentences, no code blocks, no backticks. Mention the file's purpose and notable actions. Prefer verbs. If the text is configuration, say what it configures.")
	temp := float32(0.2)
	maxTokens := int32(120)
	cfg := genai.GenerateContentConfig{
		Temperature:       &temp,
		MaxOutputTokens:   maxTokens,
		SystemInstruction: prompt[0],
	}

	userPrompt := "Path: " + filePath + "\nLanguage: " + language + "\n---\n" + content
	resp, err := c.client.Models.GenerateContent(ctx, c.config.SummaryModel, genai.Text(userPrompt), &cfg)
	if err != nil {
		return "", fmt.Errorf("summarization failed: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("no summary returned")
	}

	// Extract text from the first part
	part := resp.Candidates[0].Content.Parts[0]

	summary := strings.TrimSpace(string(part.Text))
	summary = strings.ReplaceAll(summary, "\n", " ")
	return summary, nil
}

func (c *VertexAIClient) Dim() int {
	return c.config.Dim
}
