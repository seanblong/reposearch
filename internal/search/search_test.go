package search

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/seanblong/reposearch/internal/ai"
	"github.com/seanblong/reposearch/internal/store"
	"github.com/seanblong/reposearch/pkg/models"
)

// MockAIClient implements the ai.Client interface for testing
type MockAIClient struct {
	EmbedFunc     func(text string) ([]float32, error)
	SummarizeFunc func(ctx context.Context, filePath, language, content string) (string, error)
	DimFunc       func() int
}

func (m *MockAIClient) Embed(text string) ([]float32, error) {
	if m.EmbedFunc != nil {
		return m.EmbedFunc(text)
	}
	// Default implementation returns a simple embedding
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *MockAIClient) Summarize(ctx context.Context, filePath, language, content string) (string, error) {
	if m.SummarizeFunc != nil {
		return m.SummarizeFunc(ctx, filePath, language, content)
	}
	return "mock summary", nil
}

func (m *MockAIClient) Dim() int {
	if m.DimFunc != nil {
		return m.DimFunc()
	}
	return 3
}

// MockSearchableStore implements the SearchableStore interface for testing
type MockSearchableStore struct {
	SearchFunc func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error)
}

func (m *MockSearchableStore) Search(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, head, k, opt)
	}
	return []models.SearchResult{}, nil
}

func (m *MockSearchableStore) GetRepositories(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockSearchableStore) GetChunkMeta(ctx context.Context, repository, path string, ls, le int) (store.ChunkMeta, bool, error) {
	return store.ChunkMeta{}, false, nil
}

func (m *MockSearchableStore) UpsertChunk(ctx context.Context, c models.Chunk, summaryVec []float32, contentHash string) error {
	return nil
}

func (m *MockSearchableStore) Migrate(ctx context.Context, summaryDim int) error {
	return nil
}

// TestService_Query tests the real Service.Query method with mocked dependencies
func TestService_Query(t *testing.T) {
	// Create test data
	testTime := time.Now()
	sampleChunk := models.Chunk{
		ID:         "test-chunk-1",
		Repository: "test-repo",
		Path:       "src/main.go",
		Language:   "go",
		Summary:    "Main package for testing",
		Content:    "package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}",
		LineStart:  1,
		LineEnd:    5,
		CreatedAt:  testTime,
	}

	sampleResults := []models.SearchResult{
		{
			Chunk: sampleChunk,
			Score: 0.95,
		},
	}

	tests := []struct {
		name           string
		query          string
		k              int
		opt            store.QueryOpts
		mockEmbedFunc  func(text string) ([]float32, error)
		mockSearchFunc func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error)
		expectedResult []models.SearchResult
		expectedError  error
	}{
		{
			name:  "successful query with results",
			query: "hello world function",
			k:     10,
			opt:   store.QueryOpts{Repository: "test-repo"},
			mockEmbedFunc: func(text string) ([]float32, error) {
				if text != "hello world function" {
					t.Errorf("Expected embedding text 'hello world function', got '%s'", text)
				}
				return []float32{0.1, 0.2, 0.3, 0.4, 0.5}, nil
			},
			mockSearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
				expectedVec := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
				if !reflect.DeepEqual(head, expectedVec) {
					t.Errorf("Expected head vector %v, got %v", expectedVec, head)
				}
				if k != 10 {
					t.Errorf("Expected k=10, got k=%d", k)
				}
				if opt.Repository != "test-repo" {
					t.Errorf("Expected repository 'test-repo', got '%s'", opt.Repository)
				}
				if opt.QueryText != "hello world function" {
					t.Errorf("Expected QueryText 'hello world function', got '%s'", opt.QueryText)
				}
				return sampleResults, nil
			},
			expectedResult: sampleResults,
			expectedError:  nil,
		},
		{
			name:  "query with leading and trailing whitespace",
			query: "   hello world   ",
			k:     5,
			opt:   store.QueryOpts{},
			mockEmbedFunc: func(text string) ([]float32, error) {
				if text != "hello world" {
					t.Errorf("Expected trimmed text 'hello world', got '%s'", text)
				}
				return []float32{0.1, 0.2}, nil
			},
			mockSearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
				if opt.QueryText != "hello world" {
					t.Errorf("Expected QueryText to be trimmed to 'hello world', got '%s'", opt.QueryText)
				}
				return []models.SearchResult{}, nil
			},
			expectedResult: []models.SearchResult{},
			expectedError:  nil,
		},
		{
			name:  "AI embedding error - ignored and nil vector passed",
			query: "test query",
			k:     10,
			opt:   store.QueryOpts{},
			mockEmbedFunc: func(text string) ([]float32, error) {
				return nil, errors.New("embedding service unavailable")
			},
			mockSearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
				// The original implementation ignores embedding errors and passes nil to search
				if head != nil {
					t.Errorf("Expected nil head vector when embedding fails, got %v", head)
				}
				return []models.SearchResult{}, nil
			},
			expectedResult: []models.SearchResult{},
			expectedError:  nil, // Query method ignores embed errors
		},
		{
			name:  "store search error",
			query: "test query",
			k:     10,
			opt:   store.QueryOpts{},
			mockEmbedFunc: func(text string) ([]float32, error) {
				return []float32{0.1, 0.2}, nil
			},
			mockSearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
				return nil, errors.New("database connection failed")
			},
			expectedResult: nil,
			expectedError:  errors.New("database connection failed"),
		},
		{
			name:  "query with all filter options",
			query: "python script",
			k:     20,
			opt: store.QueryOpts{
				Repository:   "my-repo",
				Language:     "python",
				PathContains: "scripts",
			},
			mockEmbedFunc: func(text string) ([]float32, error) {
				return []float32{0.5, 0.6, 0.7}, nil
			},
			mockSearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
				if opt.Repository != "my-repo" {
					t.Errorf("Expected repository 'my-repo', got '%s'", opt.Repository)
				}
				if opt.Language != "python" {
					t.Errorf("Expected language 'python', got '%s'", opt.Language)
				}
				if opt.PathContains != "scripts" {
					t.Errorf("Expected PathContains 'scripts', got '%s'", opt.PathContains)
				}
				if opt.QueryText != "python script" {
					t.Errorf("Expected QueryText 'python script', got '%s'", opt.QueryText)
				}
				return sampleResults, nil
			},
			expectedResult: sampleResults,
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockAIClient{
				EmbedFunc: tt.mockEmbedFunc,
			}

			// Create mock store
			mockStore := &MockSearchableStore{
				SearchFunc: tt.mockSearchFunc,
			}

			// Create the real Service using our constructor - this tests the actual implementation
			service := NewService(mockClient, mockStore)

			// Execute the query - this calls the actual Service.Query method from search.go
			ctx := context.Background()
			result, err := service.Query(ctx, tt.query, tt.k, tt.opt)

			// Check error expectations
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("Expected error '%v', got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("Expected error '%v', got '%v'", tt.expectedError, err)
				}
				return // Don't check results if we expected an error
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check results
			if !reflect.DeepEqual(result, tt.expectedResult) {
				t.Errorf("Expected result %+v, got %+v", tt.expectedResult, result)
			}
		})
	}
}

func TestService_Query_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		client      ai.Client
		store       store.ChunkStore
		query       string
		k           int
		opt         store.QueryOpts
		expectPanic bool
	}{
		{
			name:   "nil client causes panic",
			client: nil,
			store: &MockSearchableStore{
				SearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
					return []models.SearchResult{}, nil
				},
			},
			query:       "test",
			k:           10,
			opt:         store.QueryOpts{},
			expectPanic: true,
		},
		{
			name: "zero k value",
			client: &MockAIClient{
				EmbedFunc: func(text string) ([]float32, error) {
					return []float32{0.1}, nil
				},
			},
			store: &MockSearchableStore{
				SearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
					if k != 0 {
						t.Errorf("Expected k=0, got k=%d", k)
					}
					return []models.SearchResult{}, nil
				},
			},
			query:       "test",
			k:           0,
			opt:         store.QueryOpts{},
			expectPanic: false,
		},
		{
			name: "negative k value",
			client: &MockAIClient{
				EmbedFunc: func(text string) ([]float32, error) {
					return []float32{0.1}, nil
				},
			},
			store: &MockSearchableStore{
				SearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
					if k != -5 {
						t.Errorf("Expected k=-5, got k=%d", k)
					}
					return []models.SearchResult{}, nil
				},
			},
			query:       "test",
			k:           -5,
			opt:         store.QueryOpts{},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic but didn't get one")
					}
				}()
			}

			// Create service using the real constructor
			service := NewService(tt.client, tt.store)

			ctx := context.Background()
			_, err := service.Query(ctx, tt.query, tt.k, tt.opt)

			if !tt.expectPanic && err != nil {
				// Only report error if we weren't expecting a panic
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestService_Query_ContextCancellation(t *testing.T) {
	// Test that context cancellation is properly handled
	mockClient := &MockAIClient{
		EmbedFunc: func(text string) ([]float32, error) {
			return []float32{0.1, 0.2}, nil
		},
	}

	mockStore := &MockSearchableStore{
		SearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
			// Check if context is passed through
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return []models.SearchResult{}, nil
			}
		},
	}

	service := NewService(mockClient, mockStore)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := service.Query(ctx, "test query", 10, store.QueryOpts{})

	// Should get context cancellation error
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	} else if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestService_Query_EmptyEmbedding(t *testing.T) {
	// Test behavior when embedding returns empty vector
	mockClient := &MockAIClient{
		EmbedFunc: func(text string) ([]float32, error) {
			return []float32{}, nil // Empty embedding
		},
	}

	mockStore := &MockSearchableStore{
		SearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
			if len(head) != 0 {
				t.Errorf("Expected empty head vector, got %v", head)
			}
			return []models.SearchResult{}, nil
		},
	}

	service := NewService(mockClient, mockStore)

	ctx := context.Background()
	result, err := service.Query(ctx, "test query", 10, store.QueryOpts{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %v", result)
	}
}

func TestService_Query_LongQuery(t *testing.T) {
	// Test with a very long query string
	longQuery := strings.Repeat("test query with many words ", 1000)

	mockClient := &MockAIClient{
		EmbedFunc: func(text string) ([]float32, error) {
			if text != strings.TrimSpace(longQuery) {
				t.Error("Query text was not passed correctly to embedding")
			}
			return []float32{0.1}, nil
		},
	}

	mockStore := &MockSearchableStore{
		SearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
			if opt.QueryText != strings.TrimSpace(longQuery) {
				t.Error("Long query text was not preserved in QueryOpts")
			}
			return []models.SearchResult{}, nil
		},
	}

	service := NewService(mockClient, mockStore)

	ctx := context.Background()
	_, err := service.Query(ctx, longQuery, 10, store.QueryOpts{})

	if err != nil {
		t.Errorf("Unexpected error with long query: %v", err)
	}
}

func TestNewService(t *testing.T) {
	// Test the constructor
	mockClient := &MockAIClient{}
	mockStore := &MockSearchableStore{}

	service := NewService(mockClient, mockStore)

	if service == nil {
		t.Fatal("NewService returned nil")
	}

	if service.Client != mockClient {
		t.Error("Service client not set correctly")
	}

	if service.Store != mockStore {
		t.Error("Service store not set correctly")
	}
}

// Benchmark tests - these test the real Service.Query method performance
func BenchmarkService_Query(b *testing.B) {
	mockClient := &MockAIClient{
		EmbedFunc: func(text string) ([]float32, error) {
			return []float32{0.1, 0.2, 0.3, 0.4, 0.5}, nil
		},
	}

	mockStore := &MockSearchableStore{
		SearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
			return []models.SearchResult{}, nil
		},
	}

	service := NewService(mockClient, mockStore)

	ctx := context.Background()
	query := "test query for benchmarking"
	opt := store.QueryOpts{Repository: "test-repo"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Query(ctx, query, 10, opt)
	}
}

func BenchmarkService_Query_LongQuery(b *testing.B) {
	longQuery := strings.Repeat("complex search query with multiple terms ", 100)

	mockClient := &MockAIClient{
		EmbedFunc: func(text string) ([]float32, error) {
			return []float32{0.1, 0.2, 0.3, 0.4, 0.5}, nil
		},
	}

	mockStore := &MockSearchableStore{
		SearchFunc: func(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
			return []models.SearchResult{}, nil
		},
	}

	service := NewService(mockClient, mockStore)

	ctx := context.Background()
	opt := store.QueryOpts{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Query(ctx, longQuery, 10, opt)
	}
}
