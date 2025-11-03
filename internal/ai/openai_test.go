package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// MockTransport implements http.RoundTripper for testing
type MockTransport struct {
	mu             sync.RWMutex
	responses      map[string]*http.Response
	responseBodies map[string]string
	requests       []*http.Request
}

func NewMockTransport() *MockTransport {
	return &MockTransport{
		responses:      make(map[string]*http.Response),
		responseBodies: make(map[string]string),
		requests:       make([]*http.Request, 0),
	}
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store the request for inspection
	m.requests = append(m.requests, req)

	// Create a key based on method and URL
	key := fmt.Sprintf("%s %s", req.Method, req.URL.String())

	if respData, exists := m.responses[key]; exists {
		// Get the stored body for this response
		body := m.responseBodies[key]
		// Create a fresh response with a new body reader
		return &http.Response{
			StatusCode: respData.StatusCode,
			Status:     respData.Status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     copyHeaders(respData.Header),
		}, nil
	}

	// Default response if no mock is set up
	return &http.Response{
		StatusCode: 500,
		Status:     "500 Internal Server Error",
		Body:       io.NopCloser(strings.NewReader(`{"error": {"message": "Mock not configured"}}`)),
		Header:     make(http.Header),
	}, nil
}

func (m *MockTransport) AddResponse(method, url string, statusCode int, body string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s %s", method, url)
	m.responses[key] = &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:     make(http.Header),
	}
	m.responseBodies[key] = body
}

func (m *MockTransport) GetRequests() []*http.Request {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to avoid concurrent access issues
	requests := make([]*http.Request, len(m.requests))
	copy(requests, m.requests)
	return requests
}

func (m *MockTransport) ClearRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests = make([]*http.Request, 0)
}

// Helper function to copy HTTP headers
func copyHeaders(original http.Header) http.Header {
	copy := make(http.Header)
	for key, values := range original {
		copy[key] = make([]string, len(values))
		for i, value := range values {
			copy[key][i] = value
		}
	}
	return copy
}

// Helper function to create a client with mock transport
func createMockClient(transport *MockTransport) *OpenAIClient {
	config := &ClientConfig{
		APIKey:       "test-api-key",
		EmbedModel:   "text-embedding-3-small",
		SummaryModel: "gpt-4o-mini",
		Dim:          512,
		ProjectID:    "test-project",
	}

	client := NewOpenAIClient(config)
	client.http = &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
	}

	return client
}

// Test NewOpenAIClient
func TestNewOpenAIClient(t *testing.T) {
	tests := []struct {
		name            string
		config          *ClientConfig
		expectedEmbed   string
		expectedSummary string
	}{
		{
			name: "with all models specified",
			config: &ClientConfig{
				APIKey:       "test-key",
				EmbedModel:   "custom-embed-model",
				SummaryModel: "custom-summary-model",
				Dim:          768,
			},
			expectedEmbed:   "custom-embed-model",
			expectedSummary: "custom-summary-model",
		},
		{
			name: "with default models",
			config: &ClientConfig{
				APIKey: "test-key",
				Dim:    256,
			},
			expectedEmbed:   "text-embedding-3-small",
			expectedSummary: "gpt-4o-mini",
		},
		{
			name: "with empty embed model",
			config: &ClientConfig{
				APIKey:       "test-key",
				EmbedModel:   "",
				SummaryModel: "custom-summary",
				Dim:          1024,
			},
			expectedEmbed:   "text-embedding-3-small",
			expectedSummary: "custom-summary",
		},
		{
			name: "with empty summary model",
			config: &ClientConfig{
				APIKey:       "test-key",
				EmbedModel:   "custom-embed",
				SummaryModel: "",
				Dim:          384,
			},
			expectedEmbed:   "custom-embed",
			expectedSummary: "gpt-4o-mini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewOpenAIClient(tt.config)

			if client == nil {
				t.Fatal("Expected client instance, got nil")
			}
			if client.config.EmbedModel != tt.expectedEmbed {
				t.Errorf("Expected EmbedModel '%s', got '%s'", tt.expectedEmbed, client.config.EmbedModel)
			}
			if client.config.SummaryModel != tt.expectedSummary {
				t.Errorf("Expected SummaryModel '%s', got '%s'", tt.expectedSummary, client.config.SummaryModel)
			}
			if client.http == nil {
				t.Error("Expected HTTP client to be initialized")
			}
			if client.http.Timeout != 20*time.Second {
				t.Errorf("Expected timeout 20s, got %v", client.http.Timeout)
			}
		})
	}
}

// Test OpenAIClient.Embed method
func TestOpenAIClient_Embed(t *testing.T) {
	tests := []struct {
		name         string
		apiKey       string
		text         string
		statusCode   int
		responseBody string
		expectError  bool
		errorMsg     string
		expectedLen  int
	}{
		{
			name:        "missing API key",
			apiKey:      "",
			text:        "test text",
			expectError: true,
			errorMsg:    "PROVIDER_API_KEY unset",
		},
		{
			name:       "successful embedding",
			apiKey:     "test-key",
			text:       "test text",
			statusCode: 200,
			responseBody: `{
				"data": [
					{
						"embedding": [0.1, 0.2, 0.3, 0.4, 0.5]
					}
				]
			}`,
			expectError: false,
			expectedLen: 5,
		},
		{
			name:         "non-200 status code",
			apiKey:       "test-key",
			text:         "test text",
			statusCode:   400,
			responseBody: `{"error": {"message": "Bad request"}}`,
			expectError:  true,
			errorMsg:     "openai embedding non-200",
		},
		{
			name:         "invalid JSON response",
			apiKey:       "test-key",
			text:         "test text",
			statusCode:   200,
			responseBody: `invalid json`,
			expectError:  true,
		},
		{
			name:         "empty data array",
			apiKey:       "test-key",
			text:         "test text",
			statusCode:   200,
			responseBody: `{"data": []}`,
			expectError:  true,
			errorMsg:     "no embedding",
		},
		{
			name:         "rate limit error",
			apiKey:       "test-key",
			text:         "test text",
			statusCode:   429,
			responseBody: `{"error": {"message": "Rate limit exceeded"}}`,
			expectError:  true,
			errorMsg:     "openai embedding non-200",
		},
		{
			name:         "unauthorized error",
			apiKey:       "invalid-key",
			text:         "test text",
			statusCode:   401,
			responseBody: `{"error": {"message": "Invalid API key"}}`,
			expectError:  true,
			errorMsg:     "openai embedding non-200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewMockTransport()

			if tt.statusCode != 0 {
				transport.AddResponse("POST", "https://api.openai.com/v1/embeddings",
					tt.statusCode, tt.responseBody)
			}

			config := &ClientConfig{
				APIKey:     tt.apiKey,
				EmbedModel: "text-embedding-3-small",
				Dim:        512,
			}

			client := NewOpenAIClient(config)
			client.http = &http.Client{Transport: transport}

			embedding, err := client.Embed(tt.text)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				if embedding != nil {
					t.Errorf("Expected nil embedding when error occurs, got %v", embedding)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if len(embedding) != tt.expectedLen {
					t.Errorf("Expected embedding length %d, got %d", tt.expectedLen, len(embedding))
				}
			}

			// Verify request was made correctly (unless API key was missing)
			if tt.apiKey != "" {
				requests := transport.GetRequests()
				if len(requests) != 1 {
					t.Errorf("Expected 1 request, got %d", len(requests))
				} else {
					req := requests[0]
					if req.Method != "POST" {
						t.Errorf("Expected POST method, got %s", req.Method)
					}
					if req.URL.String() != "https://api.openai.com/v1/embeddings" {
						t.Errorf("Expected embeddings URL, got %s", req.URL.String())
					}

					// Check headers
					if req.Header.Get("Content-Type") != "application/json" {
						t.Error("Expected Content-Type header to be application/json")
					}
					if req.Header.Get("Authorization") != "Bearer "+tt.apiKey {
						t.Errorf("Expected Authorization header 'Bearer %s', got '%s'",
							tt.apiKey, req.Header.Get("Authorization"))
					}
				}
			}
		})
	}
}

// Test OpenAIClient.Summarize method
func TestOpenAIClient_Summarize(t *testing.T) {
	tests := []struct {
		name            string
		apiKey          string
		filePath        string
		language        string
		content         string
		statusCode      int
		responseBody    string
		expectError     bool
		errorMsg        string
		expectedSummary string
	}{
		{
			name:        "missing API key",
			apiKey:      "",
			filePath:    "test.go",
			language:    "go",
			content:     "package main",
			expectError: true,
			errorMsg:    "PROVIDER_API_KEY unset",
		},
		{
			name:       "successful summarization",
			apiKey:     "test-key",
			filePath:   "main.go",
			language:   "go",
			content:    "package main\n\nfunc main() {\n    fmt.Println(\"Hello World\")\n}",
			statusCode: 200,
			responseBody: `{
				"choices": [
					{
						"message": {
							"content": "Go main package that prints Hello World to console."
						}
					}
				]
			}`,
			expectError:     false,
			expectedSummary: "Go main package that prints Hello World to console.",
		},
		{
			name:       "content with newlines",
			apiKey:     "test-key",
			filePath:   "config.yaml",
			language:   "yaml",
			content:    "database:\n  host: localhost\n  port: 5432",
			statusCode: 200,
			responseBody: `{
				"choices": [
					{
						"message": {
							"content": "Configuration file that\nsets database connection parameters."
						}
					}
				]
			}`,
			expectError:     false,
			expectedSummary: "Configuration file that sets database connection parameters.",
		},
		{
			name:       "long content truncation",
			apiKey:     "test-key",
			filePath:   "large.txt",
			language:   "text",
			content:    strings.Repeat("x", 10000), // Longer than maxInput (8000)
			statusCode: 200,
			responseBody: `{
				"choices": [
					{
						"message": {
							"content": "Large text file with repeated content."
						}
					}
				]
			}`,
			expectError:     false,
			expectedSummary: "Large text file with repeated content.",
		},
		{
			name:       "API error response",
			apiKey:     "test-key",
			filePath:   "test.py",
			language:   "python",
			content:    "print('hello')",
			statusCode: 400,
			responseBody: `{
				"error": {
					"message": "Invalid request format"
				}
			}`,
			expectError: true,
			errorMsg:    "Invalid request format",
		},
		{
			name:         "non-JSON error response",
			apiKey:       "test-key",
			filePath:     "test.js",
			language:     "javascript",
			content:      "console.log('hello');",
			statusCode:   500,
			responseBody: "Internal Server Error",
			expectError:  true,
			errorMsg:     "500 Internal Server Error",
		},
		{
			name:         "empty choices array",
			apiKey:       "test-key",
			filePath:     "empty.txt",
			language:     "text",
			content:      "",
			statusCode:   200,
			responseBody: `{"choices": []}`,
			expectError:  true,
			errorMsg:     "no choices",
		},
		{
			name:         "invalid JSON response",
			apiKey:       "test-key",
			filePath:     "test.rb",
			language:     "ruby",
			content:      "puts 'hello'",
			statusCode:   200,
			responseBody: `invalid json`,
			expectError:  true,
		},
		{
			name:       "rate limit error",
			apiKey:     "test-key",
			filePath:   "test.php",
			language:   "php",
			content:    "<?php echo 'hello'; ?>",
			statusCode: 429,
			responseBody: `{
				"error": {
					"message": "Rate limit exceeded. Please try again later."
				}
			}`,
			expectError: true,
			errorMsg:    "Rate limit exceeded. Please try again later.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := NewMockTransport()

			if tt.statusCode != 0 {
				transport.AddResponse("POST", "https://api.openai.com/v1/chat/completions",
					tt.statusCode, tt.responseBody)
			}

			config := &ClientConfig{
				APIKey:       tt.apiKey,
				SummaryModel: "gpt-4o-mini",
				Dim:          512,
			}

			client := NewOpenAIClient(config)
			client.http = &http.Client{Transport: transport}

			ctx := context.Background()
			summary, err := client.Summarize(ctx, tt.filePath, tt.language, tt.content)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if summary != tt.expectedSummary {
					t.Errorf("Expected summary '%s', got '%s'", tt.expectedSummary, summary)
				}
			}

			// Verify request was made correctly (unless API key was missing)
			if tt.apiKey != "" {
				requests := transport.GetRequests()
				if len(requests) != 1 {
					t.Errorf("Expected 1 request, got %d", len(requests))
				} else {
					req := requests[0]
					if req.Method != "POST" {
						t.Errorf("Expected POST method, got %s", req.Method)
					}
					if req.URL.String() != "https://api.openai.com/v1/chat/completions" {
						t.Errorf("Expected chat completions URL, got %s", req.URL.String())
					}

					// Verify request payload for successful cases
					if !tt.expectError || tt.statusCode >= 400 {
						body, _ := io.ReadAll(req.Body)
						var payload map[string]any
						if err := json.Unmarshal(body, &payload); err == nil {
							if payload["model"] != config.SummaryModel {
								t.Errorf("Expected model '%s' in payload", config.SummaryModel)
							}
							if payload["temperature"] != 0.2 {
								t.Error("Expected temperature 0.2 in payload")
							}
							if payload["max_tokens"] != float64(120) {
								t.Error("Expected max_tokens 120 in payload")
							}
						}
					}
				}
			}
		})
	}
}

// Test context cancellation in Summarize
func TestOpenAIClient_SummarizeWithCancelledContext(t *testing.T) {
	// Create a server that simulates a slow response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a delay to allow context cancellation to take effect
		select {
		case <-r.Context().Done():
			// Context was cancelled
			return
		case <-time.After(100 * time.Millisecond):
			// If context wasn't cancelled, return a response
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"choices": [{"message": {"content": "Test summary"}}]}`)); err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
			}
		}
	}))
	defer server.Close()

	config := &ClientConfig{
		APIKey:       "test-api-key",
		SummaryModel: "gpt-4o-mini",
		Dim:          512,
	}

	client := NewOpenAIClient(config)

	// Replace the transport to redirect to our test server
	client.http.Transport = &redirectTransport{
		target: server.URL,
		orig:   nil,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Summarize(ctx, "test.go", "go", "package main")

	if err == nil {
		t.Error("Expected error due to cancelled context")
	}
	if !strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "operation was canceled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

// Test OpenAIClient.Dim method
func TestOpenAIClient_Dim(t *testing.T) {
	tests := []struct {
		name        string
		configDim   int
		expectedDim int
	}{
		{"default dimension", 512, 512},
		{"custom dimension", 1536, 1536},
		{"small dimension", 256, 256},
		{"zero dimension", 0, 1536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ClientConfig{
				APIKey: "test-key",
				Dim:    tt.configDim,
			}

			client := NewOpenAIClient(config)
			dim := client.Dim()

			if dim != tt.expectedDim {
				t.Errorf("Expected dimension %d, got %d", tt.expectedDim, dim)
			}
		})
	}
}

// Test setHeaders method
func TestOpenAIClient_setHeaders(t *testing.T) {
	tests := []struct {
		name                string
		apiKey              string
		projectID           string
		expectedAuthHeader  string
		expectProjectHeader bool
	}{
		{
			name:                "standard API key without project",
			apiKey:              "sk-1234567890",
			projectID:           "",
			expectedAuthHeader:  "Bearer sk-1234567890",
			expectProjectHeader: false,
		},
		{
			name:                "project API key with project ID",
			apiKey:              "sk-proj-1234567890",
			projectID:           "proj_test123",
			expectedAuthHeader:  "Bearer sk-proj-1234567890",
			expectProjectHeader: true,
		},
		{
			name:                "project API key without project ID",
			apiKey:              "sk-proj-1234567890",
			projectID:           "",
			expectedAuthHeader:  "Bearer sk-proj-1234567890",
			expectProjectHeader: false,
		},
		{
			name:                "standard API key with project ID",
			apiKey:              "sk-1234567890",
			projectID:           "proj_test123",
			expectedAuthHeader:  "Bearer sk-1234567890",
			expectProjectHeader: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ClientConfig{
				APIKey:    tt.apiKey,
				ProjectID: tt.projectID,
				Dim:       512,
			}

			client := NewOpenAIClient(config)

			// Create a test request
			req, _ := http.NewRequest("POST", "https://example.com", nil)

			// Call setHeaders
			client.setHeaders(req)

			// Check Content-Type header
			if req.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'",
					req.Header.Get("Content-Type"))
			}

			// Check Authorization header
			if req.Header.Get("Authorization") != tt.expectedAuthHeader {
				t.Errorf("Expected Authorization '%s', got '%s'",
					tt.expectedAuthHeader, req.Header.Get("Authorization"))
			}

			// Check OpenAI-Project header
			projectHeader := req.Header.Get("OpenAI-Project")
			if tt.expectProjectHeader {
				if projectHeader != tt.projectID {
					t.Errorf("Expected OpenAI-Project header '%s', got '%s'",
						tt.projectID, projectHeader)
				}
			} else {
				if projectHeader != "" {
					t.Errorf("Expected no OpenAI-Project header, got '%s'", projectHeader)
				}
			}
		})
	}
}

// Test HTTP client timeout behavior
func TestOpenAIClient_HTTPTimeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Small delay for testing
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"data": [{"embedding": [0.1, 0.2]}]}`)); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	config := &ClientConfig{
		APIKey:     "test-key",
		EmbedModel: "test-model",
		Dim:        512,
	}

	client := NewOpenAIClient(config)

	// Set a very short timeout
	client.http.Timeout = 1 * time.Millisecond

	// Override the URL to point to our test server
	// We'll use a custom transport that redirects the URL
	originalTransport := client.http.Transport
	client.http.Transport = &redirectTransport{
		target: server.URL,
		orig:   originalTransport,
	}

	_, err := client.Embed("test text")

	if err == nil {
		t.Error("Expected timeout error but got none")
	}
	if !strings.Contains(err.Error(), "timeout") &&
		!strings.Contains(err.Error(), "deadline exceeded") &&
		!strings.Contains(err.Error(), "Client.Timeout exceeded") &&
		!strings.Contains(err.Error(), "request canceled") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// Helper transport for redirecting requests to test server
type redirectTransport struct {
	target string
	orig   http.RoundTripper
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect OpenAI API calls to our test server
	if strings.Contains(req.URL.Host, "api.openai.com") {
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(rt.target, "http://")
	}

	if rt.orig != nil {
		return rt.orig.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// Test concurrent requests
func TestOpenAIClient_ConcurrentRequests(t *testing.T) {
	transport := NewMockTransport()
	transport.AddResponse("POST", "https://api.openai.com/v1/embeddings", 200,
		`{"data": [{"embedding": [0.1, 0.2, 0.3]}]}`)

	client := createMockClient(transport)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			embedding, err := client.Embed(fmt.Sprintf("test text %d", id))
			if err != nil {
				errors <- err
				return
			}
			if len(embedding) != 3 {
				errors <- fmt.Errorf("expected embedding length 3, got %d", len(embedding))
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent request error: %v", err)
	}

	// Verify correct number of requests were made
	requests := transport.GetRequests()
	if len(requests) != numGoroutines {
		t.Errorf("Expected %d requests, got %d", numGoroutines, len(requests))
	}
}

// Benchmark tests
func BenchmarkNewOpenAIClient(b *testing.B) {
	config := &ClientConfig{
		APIKey: "test-key",
		Dim:    512,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewOpenAIClient(config)
	}
}

func BenchmarkOpenAIClient_setHeaders(b *testing.B) {
	config := &ClientConfig{
		APIKey:    "sk-proj-test123",
		ProjectID: "proj_test",
		Dim:       512,
	}

	client := NewOpenAIClient(config)
	req, _ := http.NewRequest("POST", "https://example.com", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.setHeaders(req)
	}
}

// Test interface compliance
func TestOpenAIClient_InterfaceCompliance(t *testing.T) {
	// Verify OpenAIClient implements Client interface
	var _ Client = &OpenAIClient{}

	config := &ClientConfig{
		APIKey: "test-key",
		Dim:    512,
	}

	client := NewOpenAIClient(config)

	// Test that all interface methods are available
	if client.Dim() != 512 {
		t.Errorf("Expected Dim() to return 512, got %d", client.Dim())
	}
}

// Test edge cases and error conditions
func TestOpenAIClient_EdgeCases(t *testing.T) {
	t.Run("empty text embedding", func(t *testing.T) {
		transport := NewMockTransport()
		transport.AddResponse("POST", "https://api.openai.com/v1/embeddings", 200,
			`{"data": [{"embedding": []}]}`)

		client := createMockClient(transport)
		embedding, err := client.Embed("")

		if err != nil {
			t.Errorf("Expected no error for empty text, got: %v", err)
		}
		if len(embedding) != 0 {
			t.Errorf("Expected empty embedding array, got length %d", len(embedding))
		}
	})

	t.Run("very long text embedding", func(t *testing.T) {
		transport := NewMockTransport()
		transport.AddResponse("POST", "https://api.openai.com/v1/embeddings", 200,
			`{"data": [{"embedding": [0.1, 0.2]}]}`)

		client := createMockClient(transport)
		longText := strings.Repeat("a", 100000)

		embedding, err := client.Embed(longText)

		if err != nil {
			t.Errorf("Expected no error for long text, got: %v", err)
		}
		if len(embedding) != 2 {
			t.Errorf("Expected embedding length 2, got %d", len(embedding))
		}
	})

	t.Run("content truncation in summarize", func(t *testing.T) {
		transport := NewMockTransport()
		transport.AddResponse("POST", "https://api.openai.com/v1/chat/completions", 200,
			`{"choices": [{"message": {"content": "Summary of truncated content."}}]}`)

		client := createMockClient(transport)
		longContent := strings.Repeat("x", 10000) // Exceeds maxInput of 8000

		ctx := context.Background()
		summary, err := client.Summarize(ctx, "large.txt", "text", longContent)

		if err != nil {
			t.Errorf("Expected no error for long content, got: %v", err)
		}
		if summary != "Summary of truncated content." {
			t.Errorf("Expected specific summary, got '%s'", summary)
		}

		// Verify that the request body contains truncated content
		requests := transport.GetRequests()
		if len(requests) == 1 {
			body, _ := io.ReadAll(requests[0].Body)
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err == nil {
				messages := payload["messages"].([]interface{})
				userMsg := messages[1].(map[string]interface{})
				content := userMsg["content"].(string)

				// The content should contain truncated text (8000 chars max + metadata)
				if len(content) > 8200 { // Some buffer for metadata
					t.Errorf("Expected content to be truncated, got length %d", len(content))
				}
			}
		}
	})
}
