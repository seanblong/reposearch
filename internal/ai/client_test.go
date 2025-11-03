package ai

import (
	"context"
	"strings"
	"testing"
)

// Test Provider constants
func TestProviderConstants(t *testing.T) {
	tests := []struct {
		provider Provider
		expected string
	}{
		{ProviderOpenAI, "openai"},
		{ProviderVertexAI, "vertexai"},
		{ProviderStub, "stub"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			if string(tt.provider) != tt.expected {
				t.Errorf("Provider constant mismatch. Expected: %s, Got: %s", tt.expected, string(tt.provider))
			}
		})
	}
}

// Test ClientConfig struct
func TestClientConfig(t *testing.T) {
	config := &ClientConfig{
		APIKey:       "test-api-key",
		EmbedModel:   "test-embed-model",
		SummaryModel: "test-summary-model",
		Dim:          512,
		ProjectID:    "test-project",
		Provider:     ProviderOpenAI,
		Location:     "us-central1",
	}

	if config.APIKey != "test-api-key" {
		t.Errorf("Expected APIKey 'test-api-key', got '%s'", config.APIKey)
	}
	if config.EmbedModel != "test-embed-model" {
		t.Errorf("Expected EmbedModel 'test-embed-model', got '%s'", config.EmbedModel)
	}
	if config.SummaryModel != "test-summary-model" {
		t.Errorf("Expected SummaryModel 'test-summary-model', got '%s'", config.SummaryModel)
	}
	if config.Dim != 512 {
		t.Errorf("Expected Dim 512, got %d", config.Dim)
	}
	if config.ProjectID != "test-project" {
		t.Errorf("Expected ProjectID 'test-project', got '%s'", config.ProjectID)
	}
	if config.Provider != ProviderOpenAI {
		t.Errorf("Expected Provider 'openai', got '%s'", config.Provider)
	}
	if config.Location != "us-central1" {
		t.Errorf("Expected Location 'us-central1', got '%s'", config.Location)
	}
}

// Test NewClient function
func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *ClientConfig
		expectError bool
		errorMsg    string
		clientType  string
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "client config is required",
		},
		{
			name: "openai provider",
			config: &ClientConfig{
				Provider: ProviderOpenAI,
				APIKey:   "test-key",
				Dim:      512,
			},
			expectError: false,
			clientType:  "*ai.OpenAIClient",
		},
		{
			name: "vertexai provider",
			config: &ClientConfig{
				Provider: ProviderVertexAI,
				APIKey:   "test-key",
				Dim:      768,
			},
			expectError: false,
			clientType:  "*ai.VertexAIClient",
		},
		{
			name: "stub provider",
			config: &ClientConfig{
				Provider: ProviderStub,
				Dim:      256,
			},
			expectError: false,
			clientType:  "*ai.StubClient",
		},
		{
			name: "unsupported provider",
			config: &ClientConfig{
				Provider: Provider("unsupported"),
				Dim:      512,
			},
			expectError: true,
			errorMsg:    "unsupported provider: unsupported",
		},
		{
			name: "empty provider",
			config: &ClientConfig{
				Provider: Provider(""),
				Dim:      512,
			},
			expectError: true,
			errorMsg:    "unsupported provider: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				if client != nil {
					t.Errorf("Expected nil client when error occurs, got %v", client)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if client == nil {
					t.Errorf("Expected client instance, got nil")
				}
				// Check client type
				clientTypeName := ""
				switch client.(type) {
				case *OpenAIClient:
					clientTypeName = "*ai.OpenAIClient"
				case *VertexAIClient:
					clientTypeName = "*ai.VertexAIClient"
				case *StubClient:
					clientTypeName = "*ai.StubClient"
				default:
					clientTypeName = "unknown"
				}
				if clientTypeName != tt.clientType {
					t.Errorf("Expected client type '%s', got '%s'", tt.clientType, clientTypeName)
				}
			}
		})
	}
}

// Test StubClient creation
func TestNewStubClient(t *testing.T) {
	tests := []struct {
		name string
		dim  int
	}{
		{"default dimension", 512},
		{"small dimension", 128},
		{"large dimension", 1536},
		{"zero dimension", 0},
		{"negative dimension", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewStubClient(tt.dim)

			// NewStubClient always returns a valid instance
			if client.dim != tt.dim {
				t.Errorf("Expected dimension %d, got %d", tt.dim, client.dim)
			}
			if client.Dim() != tt.dim {
				t.Errorf("Expected Dim() to return %d, got %d", tt.dim, client.Dim())
			}
		})
	}
}

// Test StubClient Embed method
func TestStubClient_Embed(t *testing.T) {
	tests := []struct {
		name string
		dim  int
		text string
	}{
		{"empty text", 512, ""},
		{"short text", 256, "hello"},
		{"long text", 768, "This is a longer text that should still return a valid embedding vector"},
		{"multiline text", 384, "Line 1\nLine 2\nLine 3"},
		{"special characters", 128, "Hello! @#$%^&*()"},
		{"unicode text", 512, "Hello 世界 🌍"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewStubClient(tt.dim)
			embedding, err := client.Embed(tt.text)

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if len(embedding) != tt.dim {
				t.Errorf("Expected embedding length %d, got %d", tt.dim, len(embedding))
			}
			// Verify all values are zero (since it's a stub)
			for i, val := range embedding {
				if val != 0.0 {
					t.Errorf("Expected all embedding values to be 0.0, got %f at index %d", val, i)
				}
			}
		})
	}
}

// Test StubClient Summarize method
func TestStubClient_Summarize(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		language string
		content  string
		expected string
	}{
		{
			name:     "file with comment header",
			filePath: "test.go",
			language: "go",
			content:  "// This is a Go file\npackage main\n\nfunc main() {}",
			expected: "// This is a Go file",
		},
		{
			name:     "file with markdown header",
			filePath: "README.md",
			language: "markdown",
			content:  "# Project Title\n\nThis is a description",
			expected: "# Project Title",
		},
		{
			name:     "file with short comment",
			filePath: "test.js",
			language: "javascript",
			content:  "// Short\nfunction test() {}",
			expected: "Code file: test.js",
		},
		{
			name:     "file without comments",
			filePath: "main.py",
			language: "python",
			content:  "def main():\n    print('hello')",
			expected: "Code file: main.py",
		},
		{
			name:     "empty file",
			filePath: "empty.txt",
			language: "text",
			content:  "",
			expected: "Code file: empty.txt",
		},
		{
			name:     "file with multiple comments",
			filePath: "config.yaml",
			language: "yaml",
			content:  "# Configuration file\n# This sets up the application\nkey: value",
			expected: "# Configuration file",
		},
		{
			name:     "file with whitespace and comments",
			filePath: "script.sh",
			language: "bash",
			content:  "\n\n# Shell script for automation\n#!/bin/bash\necho 'hello'",
			expected: "# Shell script for automation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewStubClient(512)
			ctx := context.Background()

			summary, err := client.Summarize(ctx, tt.filePath, tt.language, tt.content)

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if summary != tt.expected {
				t.Errorf("Expected summary '%s', got '%s'", tt.expected, summary)
			}
		})
	}
}

// Test StubClient Summarize with context cancellation
func TestStubClient_SummarizeWithCancelledContext(t *testing.T) {
	client := NewStubClient(512)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should still work because StubClient doesn't check context
	summary, err := client.Summarize(ctx, "test.go", "go", "// Test file\npackage main")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if summary != "// Test file" {
		t.Errorf("Expected summary '// Test file', got '%s'", summary)
	}
}

// Test min helper function
func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a smaller", 3, 5, 3},
		{"b smaller", 7, 2, 2},
		{"equal values", 4, 4, 4},
		{"negative values", -3, -1, -3},
		{"zero and positive", 0, 5, 0},
		{"zero and negative", 0, -2, -2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("min(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Test Client interface compliance
func TestClientInterfaceCompliance(t *testing.T) {
	// Test that StubClient implements Client interface
	var _ Client = &StubClient{}

	// Test that the interface methods work as expected
	client := NewStubClient(256)

	// Test Embed method
	embedding, err := client.Embed("test")
	if err != nil {
		t.Errorf("Expected no error from Embed, got: %v", err)
	}
	if len(embedding) != 256 {
		t.Errorf("Expected embedding length 256, got %d", len(embedding))
	}

	// Test Summarize method
	ctx := context.Background()
	summary, err := client.Summarize(ctx, "test.go", "go", "// Test\npackage main")
	if err != nil {
		t.Errorf("Expected no error from Summarize, got: %v", err)
	}
	if summary == "" {
		t.Errorf("Expected non-empty summary")
	}

	// Test Dim method
	if client.Dim() != 256 {
		t.Errorf("Expected Dim() to return 256, got %d", client.Dim())
	}
}

// Benchmark tests
func BenchmarkNewStubClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewStubClient(512)
	}
}

func BenchmarkStubClient_Embed(b *testing.B) {
	client := NewStubClient(512)
	text := "This is a test text for embedding benchmark"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Embed(text)
	}
}

func BenchmarkStubClient_Summarize(b *testing.B) {
	client := NewStubClient(512)
	ctx := context.Background()
	content := "# Test File\n// This is a test file\npackage main\n\nfunc main() {\n    fmt.Println(\"hello\")\n}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Summarize(ctx, "test.go", "go", content)
	}
}

func BenchmarkNewClient(b *testing.B) {
	config := &ClientConfig{
		Provider: ProviderStub,
		Dim:      512,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewClient(config)
	}
}

// Test edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	t.Run("StubClient with very large dimension", func(t *testing.T) {
		client := NewStubClient(100000)
		embedding, err := client.Embed("test")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if len(embedding) != 100000 {
			t.Errorf("Expected embedding length 100000, got %d", len(embedding))
		}
	})

	t.Run("Summarize with very long content", func(t *testing.T) {
		client := NewStubClient(512)
		ctx := context.Background()

		// Create very long content
		longContent := strings.Repeat("line\n", 1000)
		longContent = "# Long file header\n" + longContent

		summary, err := client.Summarize(ctx, "long.txt", "text", longContent)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if summary != "# Long file header" {
			t.Errorf("Expected summary '# Long file header', got '%s'", summary)
		}
	})

	t.Run("Provider type conversion", func(t *testing.T) {
		provider := Provider("custom")
		if string(provider) != "custom" {
			t.Errorf("Expected string conversion 'custom', got '%s'", string(provider))
		}
	})
}

// Test concurrent access to StubClient
func TestStubClientConcurrency(t *testing.T) {
	client := NewStubClient(512)
	ctx := context.Background()

	// Test concurrent Embed calls
	t.Run("concurrent embeds", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()

				embedding, err := client.Embed("test text")
				if err != nil {
					t.Errorf("Goroutine %d: Expected no error, got: %v", id, err)
				}
				if len(embedding) != 512 {
					t.Errorf("Goroutine %d: Expected embedding length 512, got %d", id, len(embedding))
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	// Test concurrent Summarize calls
	t.Run("concurrent summarizes", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()

				summary, err := client.Summarize(ctx, "test.go", "go", "// Test file\npackage main")
				if err != nil {
					t.Errorf("Goroutine %d: Expected no error, got: %v", id, err)
				}
				if summary != "// Test file" {
					t.Errorf("Goroutine %d: Expected summary '// Test file', got '%s'", id, summary)
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
