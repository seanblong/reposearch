package ai

import (
	"context"
	"errors"
	"strings"
)

// Client provides both embedding and summarization capabilities
type Client interface {
	Embed(text string) ([]float32, error)
	Summarize(ctx context.Context, filePath, language, content string) (string, error)
	Dim() int
}

// Provider is enumeration of supported AI providers
type Provider string

const (
	ProviderOpenAI   Provider = "openai"
	ProviderVertexAI Provider = "vertexai"
	ProviderStub     Provider = "stub"
)

// ClientConfig holds configuration for AI clients
type ClientConfig struct {
	APIKey       string
	EmbedModel   string
	SummaryModel string
	Dim          int
	ProjectID    string
	Provider     Provider
	Location     string
}

// NewClient creates a new AI client based on configuration
func NewClient(config *ClientConfig) (Client, error) {
	if config == nil {
		return nil, errors.New("client config is required")
	}

	ctx := context.Background()
	switch config.Provider {
	case ProviderOpenAI:
		return NewOpenAIClient(config), nil
	case ProviderVertexAI:
		return NewVertexAIClient(ctx, config)
	case ProviderStub:
		return NewStubClient(config.Dim), nil
	default:
		return nil, errors.New("unsupported provider: " + string(config.Provider))
	}
}

// StubClient is a stub implementation of the Client interface for testing
type StubClient struct {
	dim int
}

// NewStubClient creates a new StubClient
func NewStubClient(dim int) *StubClient {
	return &StubClient{dim: dim}
}

// Embed implements the embedding functionality
func (s *StubClient) Embed(text string) ([]float32, error) {
	return make([]float32, s.dim), nil
}

// Summarize implements the summarization functionality
func (s *StubClient) Summarize(ctx context.Context, filePath, language, content string) (string, error) {
	// Simple heuristic summary for testing
	lines := strings.Split(content, "\n")
	for _, line := range lines[:min(5, len(lines))] {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			if len(line) > 10 {
				return line, nil
			}
		}
	}
	return "Code file: " + filePath, nil
}

// Dim returns the embedding dimension
func (s *StubClient) Dim() int {
	return s.dim
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
