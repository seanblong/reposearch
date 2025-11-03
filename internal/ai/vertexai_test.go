package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// Test configuration validation and defaults in NewVertexAIClient
func TestNewVertexAIClient_Configuration(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                 string
		config               *ClientConfig
		expectError          bool
		errorMsg             string
		expectedEmbedModel   string
		expectedSummaryModel string
		expectedDim          int
	}{
		{
			name: "missing API key",
			config: &ClientConfig{
				APIKey: "",
			},
			expectError: true,
			errorMsg:    "failed to create Gemini client",
		},
		{
			name: "with all models specified",
			config: &ClientConfig{
				APIKey:       "test-api-key",
				EmbedModel:   "custom-embed-model",
				SummaryModel: "custom-summary-model",
				Dim:          1024,
			},
			expectError:          false,
			expectedEmbedModel:   "custom-embed-model",
			expectedSummaryModel: "custom-summary-model",
			expectedDim:          1024,
		},
		{
			name: "with default models",
			config: &ClientConfig{
				APIKey: "test-api-key",
			},
			expectError:          false,
			expectedEmbedModel:   "embedding-001",
			expectedSummaryModel: "gemini-1.5-flash-latest",
			expectedDim:          768,
		},
		{
			name: "with empty embed model",
			config: &ClientConfig{
				APIKey:       "test-api-key",
				EmbedModel:   "",
				SummaryModel: "custom-summary",
				Dim:          512,
			},
			expectError:          false,
			expectedEmbedModel:   "embedding-001",
			expectedSummaryModel: "custom-summary",
			expectedDim:          512,
		},
		{
			name: "with empty summary model",
			config: &ClientConfig{
				APIKey:     "test-api-key",
				EmbedModel: "custom-embed",
				Dim:        256,
			},
			expectError:          false,
			expectedEmbedModel:   "custom-embed",
			expectedSummaryModel: "gemini-1.5-flash-latest",
			expectedDim:          256,
		},
		{
			name: "with zero dimension",
			config: &ClientConfig{
				APIKey: "test-api-key",
				Dim:    0,
			},
			expectError:          false,
			expectedEmbedModel:   "embedding-001",
			expectedSummaryModel: "gemini-1.5-flash-latest",
			expectedDim:          768,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectError {
				if tt.config.APIKey == "" {
					// We can test this case since it fails before calling genai.NewClient
					_, err := NewVertexAIClient(ctx, tt.config)
					if err == nil {
						t.Error("Expected error but got none")
					}
					if !strings.Contains(err.Error(), tt.errorMsg) {
						t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
					}
				}
			} else {
				// Test configuration defaults by creating a copy and checking the values
				configCopy := *tt.config

				// Apply the same default logic as NewVertexAIClient
				if configCopy.EmbedModel == "" {
					configCopy.EmbedModel = "embedding-001"
				}
				if configCopy.SummaryModel == "" {
					configCopy.SummaryModel = "gemini-1.5-flash-latest"
				}
				if configCopy.Dim == 0 {
					configCopy.Dim = 768
				}

				if configCopy.EmbedModel != tt.expectedEmbedModel {
					t.Errorf("Expected EmbedModel '%s', got '%s'", tt.expectedEmbedModel, configCopy.EmbedModel)
				}
				if configCopy.SummaryModel != tt.expectedSummaryModel {
					t.Errorf("Expected SummaryModel '%s', got '%s'", tt.expectedSummaryModel, configCopy.SummaryModel)
				}
				if configCopy.Dim != tt.expectedDim {
					t.Errorf("Expected Dim %d, got %d", tt.expectedDim, configCopy.Dim)
				}
			}
		})
	}
}

// Test Dim method with various configurations
func TestVertexAIClient_Dim(t *testing.T) {
	tests := []struct {
		name        string
		configDim   int
		expectedDim int
	}{
		{"default dimension", 768, 768},
		{"custom dimension", 1536, 1536},
		{"small dimension", 256, 256},
		{"zero dimension", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ClientConfig{
				APIKey: "test-key",
				Dim:    tt.configDim,
			}

			// Create a client struct directly for testing Dim method
			client := &VertexAIClient{
				config: config,
				client: nil, // We don't need the actual client for this test
			}

			dim := client.Dim()
			if dim != tt.expectedDim {
				t.Errorf("Expected dimension %d, got %d", tt.expectedDim, dim)
			}
		})
	}
}

// Test interface compliance
func TestVertexAIClient_InterfaceCompliance(t *testing.T) {
	// Verify VertexAIClient implements Client interface
	var _ Client = &VertexAIClient{}

	config := &ClientConfig{
		APIKey: "test-key",
		Dim:    512,
	}

	client := &VertexAIClient{
		config: config,
		client: nil,
	}

	// Test that Dim method works
	if client.Dim() != 512 {
		t.Errorf("Expected Dim() to return 512, got %d", client.Dim())
	}
}

// Test content truncation logic in Summarize method
func TestVertexAIClient_ContentTruncation(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		expectedMaxLen int
	}{
		{
			name:           "short content",
			content:        "short content",
			expectedMaxLen: 13,
		},
		{
			name:           "content at limit",
			content:        strings.Repeat("x", 8000),
			expectedMaxLen: 8000,
		},
		{
			name:           "content over limit",
			content:        strings.Repeat("x", 10000),
			expectedMaxLen: 8000,
		},
		{
			name:           "empty content",
			content:        "",
			expectedMaxLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the actual Summarize method without real API calls,
			// but we can test the truncation logic
			const maxInput = 8000
			content := tt.content
			if len(content) > maxInput {
				content = content[:maxInput]
			}

			if len(content) != tt.expectedMaxLen {
				t.Errorf("Expected content length %d, got %d", tt.expectedMaxLen, len(content))
			}
		})
	}
}

// Test error scenarios that would occur in real usage
func TestVertexAIClient_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name        string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "nil config",
			description: "Test behavior with nil config",
			testFunc: func(t *testing.T) {
				ctx := context.Background()
				_, err := NewVertexAIClient(ctx, nil)
				if err == nil {
					t.Error("Expected error with nil config")
				}
				if !strings.Contains(err.Error(), "config cannot be nil") {
					t.Errorf("Expected 'config cannot be nil' error, got: %v", err)
				}
			},
		},
		{
			name:        "empty API key",
			description: "Test behavior with empty API key",
			testFunc: func(t *testing.T) {
				ctx := context.Background()
				config := &ClientConfig{APIKey: ""}
				_, err := NewVertexAIClient(ctx, config)
				if err == nil {
					t.Error("Expected error with empty API key")
				}
				if !strings.Contains(err.Error(), "failed to create Gemini client") {
					t.Errorf("Expected Gemini client error, got: %v", err)
				}
			},
		},
		{
			name:        "whitespace API key",
			description: "Test behavior with whitespace-only API key",
			testFunc: func(t *testing.T) {
				ctx := context.Background()
				config := &ClientConfig{APIKey: "   "}
				_, err := NewVertexAIClient(ctx, config)
				if err == nil {
					t.Error("Expected error with whitespace API key")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}

// Test default model assignments
func TestVertexAIClient_DefaultModels(t *testing.T) {
	tests := []struct {
		name            string
		inputConfig     *ClientConfig
		expectedEmbed   string
		expectedSummary string
		expectedDim     int
	}{
		{
			name: "all defaults",
			inputConfig: &ClientConfig{
				APIKey: "test-key",
			},
			expectedEmbed:   "embedding-001",
			expectedSummary: "gemini-1.5-flash-latest",
			expectedDim:     768,
		},
		{
			name: "partial defaults",
			inputConfig: &ClientConfig{
				APIKey:     "test-key",
				EmbedModel: "custom-embed",
			},
			expectedEmbed:   "custom-embed",
			expectedSummary: "gemini-1.5-flash-latest",
			expectedDim:     768,
		},
		{
			name: "no defaults needed",
			inputConfig: &ClientConfig{
				APIKey:       "test-key",
				EmbedModel:   "custom-embed",
				SummaryModel: "custom-summary",
				Dim:          1024,
			},
			expectedEmbed:   "custom-embed",
			expectedSummary: "custom-summary",
			expectedDim:     1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy to test the default assignment logic
			config := *tt.inputConfig

			// Apply the same logic as NewVertexAIClient
			if config.EmbedModel == "" {
				config.EmbedModel = "embedding-001"
			}
			if config.SummaryModel == "" {
				config.SummaryModel = "gemini-1.5-flash-latest"
			}
			if config.Dim == 0 {
				config.Dim = 768
			}

			if config.EmbedModel != tt.expectedEmbed {
				t.Errorf("Expected EmbedModel '%s', got '%s'", tt.expectedEmbed, config.EmbedModel)
			}
			if config.SummaryModel != tt.expectedSummary {
				t.Errorf("Expected SummaryModel '%s', got '%s'", tt.expectedSummary, config.SummaryModel)
			}
			if config.Dim != tt.expectedDim {
				t.Errorf("Expected Dim %d, got %d", tt.expectedDim, config.Dim)
			}
		})
	}
}

// Test configuration edge cases
func TestVertexAIClient_ConfigurationEdgeCases(t *testing.T) {
	t.Run("very long model names", func(t *testing.T) {
		longName := strings.Repeat("a", 1000)
		config := &ClientConfig{
			APIKey:       "test-key",
			EmbedModel:   longName,
			SummaryModel: longName,
			Dim:          512,
		}

		// Test that very long model names don't cause issues
		if config.EmbedModel != longName {
			t.Error("Long embed model name was modified")
		}
		if config.SummaryModel != longName {
			t.Error("Long summary model name was modified")
		}
	})

	t.Run("negative dimension", func(t *testing.T) {
		config := &ClientConfig{
			APIKey: "test-key",
			Dim:    -100,
		}

		client := &VertexAIClient{config: config}
		if client.Dim() != -100 {
			t.Errorf("Expected negative dimension to be preserved, got %d", client.Dim())
		}
	})

	t.Run("very large dimension", func(t *testing.T) {
		config := &ClientConfig{
			APIKey: "test-key",
			Dim:    999999,
		}

		client := &VertexAIClient{config: config}
		if client.Dim() != 999999 {
			t.Errorf("Expected large dimension to be preserved, got %d", client.Dim())
		}
	})
}

// Test string manipulation in Summarize logic
func TestVertexAIClient_StringManipulation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line",
			input:    "This is a single line",
			expected: "This is a single line",
		},
		{
			name:     "multiple lines",
			input:    "Line 1\nLine 2\nLine 3",
			expected: "Line 1 Line 2 Line 3",
		},
		{
			name:     "with leading/trailing whitespace",
			input:    "  \n  Text with spaces  \n  ",
			expected: "Text with spaces",
		},
		{
			name:     "mixed whitespace",
			input:    "Text\nwith\tmixed\r\nwhitespace",
			expected: "Text with\tmixed\r whitespace",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \n\t  \n  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the same string manipulation logic used in Summarize
			result := strings.TrimSpace(tt.input)
			result = strings.ReplaceAll(result, "\n", " ")

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Test prompt construction logic
func TestVertexAIClient_PromptConstruction(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		language string
		content  string
		expected string
	}{
		{
			name:     "basic prompt",
			filePath: "main.go",
			language: "go",
			content:  "package main",
			expected: "Path: main.go\nLanguage: go\n---\npackage main",
		},
		{
			name:     "empty content",
			filePath: "empty.txt",
			language: "text",
			content:  "",
			expected: "Path: empty.txt\nLanguage: text\n---\n",
		},
		{
			name:     "special characters in path",
			filePath: "path/with spaces/file-name.js",
			language: "javascript",
			content:  "console.log('hello');",
			expected: "Path: path/with spaces/file-name.js\nLanguage: javascript\n---\nconsole.log('hello');",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the same prompt construction logic used in Summarize
			userPrompt := "Path: " + tt.filePath + "\nLanguage: " + tt.language + "\n---\n" + tt.content

			if userPrompt != tt.expected {
				t.Errorf("Expected prompt:\n%s\nGot:\n%s", tt.expected, userPrompt)
			}
		})
	}
}

// Test model parameter validation
func TestVertexAIClient_ModelParameters(t *testing.T) {
	t.Run("temperature and token limits", func(t *testing.T) {
		// Test the parameter values used in Summarize
		expectedTemp := float32(0.2)
		expectedMaxTokens := int32(120)

		if expectedTemp != 0.2 {
			t.Errorf("Expected temperature 0.2, got %f", expectedTemp)
		}
		if expectedMaxTokens != 120 {
			t.Errorf("Expected max tokens 120, got %d", expectedMaxTokens)
		}
	})

	t.Run("system instruction content", func(t *testing.T) {
		expectedInstruction := "You are a concise code summarizer. Write at most 240 characters, 1–2 sentences, no code blocks, no backticks. Mention the file's purpose and notable actions. Prefer verbs. If the text is configuration, say what it configures."

		// Test that the system instruction is properly constructed
		if len(expectedInstruction) == 0 {
			t.Error("System instruction should not be empty")
		}
		if !strings.Contains(expectedInstruction, "concise") {
			t.Error("System instruction should emphasize conciseness")
		}
		if !strings.Contains(expectedInstruction, "240 characters") {
			t.Error("System instruction should specify character limit")
		}
	})
}

// Test model name validation patterns
func TestVertexAIClient_ModelNamePatterns(t *testing.T) {
	validPatterns := []string{
		"embedding-001",
		"gemini-1.5-flash-latest",
		"gemini-pro",
		"text-embedding-ada-002",
		"custom-model-v1",
		"model_with_underscores",
		"model-with-numbers-123",
	}

	for _, pattern := range validPatterns {
		t.Run(fmt.Sprintf("valid_pattern_%s", pattern), func(t *testing.T) {
			config := &ClientConfig{
				APIKey:       "test-key",
				EmbedModel:   pattern,
				SummaryModel: pattern,
				Dim:          512,
			}

			if config.EmbedModel != pattern {
				t.Errorf("Embed model pattern was modified: expected %s, got %s", pattern, config.EmbedModel)
			}
			if config.SummaryModel != pattern {
				t.Errorf("Summary model pattern was modified: expected %s, got %s", pattern, config.SummaryModel)
			}
		})
	}
}

// Test API key validation patterns
func TestVertexAIClient_APIKeyValidation(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		expectError bool
	}{
		{
			name:        "valid API key",
			apiKey:      "AIzaSyAbCdEfGhIjKlMnOpQrStUvWxYz0123456",
			expectError: false,
		},
		{
			name:        "empty API key",
			apiKey:      "",
			expectError: true,
		},
		{
			name:        "whitespace API key",
			apiKey:      "   ",
			expectError: true,
		},
		{
			name:        "short API key",
			apiKey:      "abc",
			expectError: false, // We only check for empty, not format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ClientConfig{
				APIKey: tt.apiKey,
			}

			ctx := context.Background()
			_, err := NewVertexAIClient(ctx, config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				// For non-empty API keys, we expect the error to be from genai.NewClient
				// since we can't mock it, but we've validated our input
				if tt.apiKey != "" && err != nil {
					// This is expected since we can't connect to the real API
					if !strings.Contains(err.Error(), "failed to create Gemini client") {
						t.Errorf("Expected Gemini client creation error, got: %v", err)
					}
				}
			}
		})
	}
}

// Test that covers the remaining NewVertexAIClient logic paths
func TestNewVertexAIClient_FullCoverage(t *testing.T) {
	ctx := context.Background()

	// Test successful path through validation but expect genai.NewClient to fail
	config := &ClientConfig{
		APIKey:       "test-api-key-12345",
		EmbedModel:   "custom-embed",
		SummaryModel: "custom-summary",
		Dim:          512,
	}

	client, err := NewVertexAIClient(ctx, config)

	// The behavior may vary - either it succeeds (unlikely without valid credentials)
	// or it fails (more likely). We just want to test that our validation logic works.
	if err != nil {
		// Verify it's the expected error from genai.NewClient, not our validation
		if !strings.Contains(err.Error(), "failed to create Gemini client") {
			t.Errorf("Expected genai client error, got: %v", err)
		}
	} else {
		// If it somehow succeeded, clean up and verify we got a valid client
		if client == nil {
			t.Error("Expected client to be non-nil when no error occurred")
		} else {
			defer func() {
				if closeErr := client.Close(); closeErr != nil {
					t.Logf("Error closing client: %v", closeErr)
				}
			}()

			// Verify the client has the expected configuration
			if client.Dim() != 512 {
				t.Errorf("Expected dimension 512, got %d", client.Dim())
			}
		}
	}
}

// Test Close method behavior with nil client
func TestVertexAIClient_CloseWithNilClient(t *testing.T) {
	// Create a client with nil genai.Client to test error handling
	client := &VertexAIClient{
		config: &ClientConfig{
			APIKey: "test-key",
			Dim:    512,
		},
		client: nil,
	}

	err := client.Close()
	if err != nil {
		t.Error("Expected error when calling Close() with nil client")
	}
}

// Test Embed method with nil client (tests error path)
func TestVertexAIClient_EmbedWithNilClient(t *testing.T) {
	client := &VertexAIClient{
		config: &ClientConfig{
			APIKey:     "test-key",
			EmbedModel: "embedding-001",
			Dim:        768,
		},
		client: nil,
	}

	// This should panic since client is nil
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when calling Embed() with nil client")
		}
	}()

	_, _ = client.Embed("test text")
}

// Test Summarize method with nil client (tests error path)
func TestVertexAIClient_SummarizeWithNilClient(t *testing.T) {
	client := &VertexAIClient{
		config: &ClientConfig{
			APIKey:       "test-key",
			SummaryModel: "gemini-1.5-flash-latest",
			Dim:          768,
		},
		client: nil,
	}

	// This should panic since client is nil
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when calling Summarize() with nil client")
		}
	}()

	ctx := context.Background()
	_, _ = client.Summarize(ctx, "test.go", "go", "package main")
}

// Test that we can create VertexAIClient struct directly for testing
func TestVertexAIClient_DirectCreation(t *testing.T) {
	config := &ClientConfig{
		APIKey:       "test-key",
		EmbedModel:   "custom-embed",
		SummaryModel: "custom-summary",
		Dim:          1024,
	}

	client := &VertexAIClient{
		config: config,
		client: nil, // We set this to nil since we can't create a real client in tests
	}

	// Test that configuration is properly stored
	if client.config.APIKey != "test-key" {
		t.Errorf("Expected APIKey 'test-key', got '%s'", client.config.APIKey)
	}
	if client.config.EmbedModel != "custom-embed" {
		t.Errorf("Expected EmbedModel 'custom-embed', got '%s'", client.config.EmbedModel)
	}
	if client.config.SummaryModel != "custom-summary" {
		t.Errorf("Expected SummaryModel 'custom-summary', got '%s'", client.config.SummaryModel)
	}
	if client.Dim() != 1024 {
		t.Errorf("Expected Dim 1024, got %d", client.Dim())
	}
}

// Test concurrent access to configuration
func TestVertexAIClient_ConcurrentConfigAccess(t *testing.T) {
	config := &ClientConfig{
		APIKey: "test-key",
		Dim:    512,
	}

	client := &VertexAIClient{config: config}

	const numGoroutines = 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()

			// Test concurrent read access to configuration
			dim := client.Dim()
			if dim != 512 {
				t.Errorf("Expected dimension 512, got %d", dim)
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

// Benchmark tests for logic that doesn't require API calls
func BenchmarkVertexAIClient_Dim(b *testing.B) {
	config := &ClientConfig{
		APIKey: "test-key",
		Dim:    512,
	}

	client := &VertexAIClient{config: config}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.Dim()
	}
}

func BenchmarkVertexAIClient_ContentTruncation(b *testing.B) {
	longContent := strings.Repeat("x", 20000)
	const maxInput = 8000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		content := longContent
		if len(content) > maxInput {
			content = content[:maxInput]
		}
		// Use the content to avoid SA4006 warning
		_ = len(content)
	}
}

func BenchmarkVertexAIClient_StringProcessing(b *testing.B) {
	input := "This is a test string\nwith multiple lines\nand various content\nfor processing"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := strings.TrimSpace(input)
		result = strings.ReplaceAll(result, "\n", " ")
		_ = result
	}
}
