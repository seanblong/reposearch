package indexer

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/karrick/godirwalk"
	"github.com/rs/zerolog"
	"github.com/seanblong/reposearch/internal/ai"
	"github.com/seanblong/reposearch/internal/store"
	"github.com/seanblong/reposearch/pkg/models"
)

func init() {
	// Suppress logs during testing
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

// MockIndexableStore implements IndexableStore for testing
type MockIndexableStore struct {
	GetChunkMetaFunc func(ctx context.Context, repository, path string, ls, le int) (store.ChunkMeta, bool, error)
	UpsertChunkFunc  func(ctx context.Context, c models.Chunk, summaryVec []float32, contentHash string) error
}

func (m *MockIndexableStore) Search(ctx context.Context, head []float32, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
	return []models.SearchResult{}, nil
}

func (m *MockIndexableStore) GetRepositories(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockIndexableStore) Migrate(ctx context.Context, summaryDim int) error {
	return nil
}

func (m *MockIndexableStore) GetChunkMeta(ctx context.Context, repository, path string, ls, le int) (store.ChunkMeta, bool, error) {
	if m.GetChunkMetaFunc != nil {
		return m.GetChunkMetaFunc(ctx, repository, path, ls, le)
	}
	return store.ChunkMeta{}, false, nil
}

func (m *MockIndexableStore) UpsertChunk(ctx context.Context, c models.Chunk, summaryVec []float32, contentHash string) error {
	if m.UpsertChunkFunc != nil {
		return m.UpsertChunkFunc(ctx, c, summaryVec, contentHash)
	}
	return nil
}

// MockAIClient implements ai.Client for testing
type MockAIClient struct {
	EmbedFunc     func(text string) ([]float32, error)
	SummarizeFunc func(ctx context.Context, filePath, language, content string) (string, error)
	DimFunc       func() int
}

func (m *MockAIClient) Embed(text string) ([]float32, error) {
	if m.EmbedFunc != nil {
		return m.EmbedFunc(text)
	}
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

// MockFileSystemWalker implements FileSystemWalker for testing
type MockFileSystemWalker struct {
	FilesToProcess []string // List of file paths to process
	WalkError      error    // Error to return from Walk
}

func (m *MockFileSystemWalker) Walk(root string, options *godirwalk.Options) error {
	if m.WalkError != nil {
		return m.WalkError
	}

	// For testing, we'll bypass the actual godirwalk.Dirent complexity
	// and just process our mock files directly
	for _, filePath := range m.FilesToProcess {
		// Skip files that should be skipped according to indexer logic
		if shouldSkip(filePath) {
			continue
		}

		// We need to simulate the callback but we can't easily mock godirwalk.Dirent
		// So let's create a custom test walker function that calls our test logic directly
		// Rather than trying to mock the Dirent, let's call our test file processing directly

		// We'll simulate what the callback would do by calling our file reader
		// This bypasses the Dirent issue entirely
		err := m.simulateFileProcessing(filePath, options, root)
		if err != nil {
			return err
		}
	}
	return nil
}

// simulateFileProcessing simulates what the indexer callback would do
func (m *MockFileSystemWalker) simulateFileProcessing(filePath string, options *godirwalk.Options, root string) error {
	// This is a bit of a hack, but we'll call the callback with a nil Dirent
	// and modify the indexer to handle this case in tests
	return options.Callback(filePath, nil)
}

// MockFileReader implements FileReader for testing
type MockFileReader struct {
	ReadFileFunc func(filename string) ([]byte, error)
	Files        map[string]string // path -> content
}

func (m *MockFileReader) ReadFile(filename string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(filename)
	}

	// Default implementation returns content from Files map
	if content, exists := m.Files[filename]; exists {
		return []byte(content), nil
	}
	return nil, errors.New("file not found")
}

func TestIndexer_Run(t *testing.T) {
	tests := []struct {
		name            string
		repoRoot        string
		repository      string
		files           map[string]string // path -> content
		mockStore       *MockIndexableStore
		mockClient      *MockAIClient
		expectedError   error
		validateResults func(t *testing.T, store *MockIndexableStore, client *MockAIClient)
	}{
		{
			name:       "successful indexing of single go file",
			repoRoot:   "/test/repo",
			repository: "test/repo",
			files: map[string]string{
				"/test/repo/main.go": "package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}",
			},
			mockStore: &MockIndexableStore{
				GetChunkMetaFunc: func(ctx context.Context, repository, path string, ls, le int) (store.ChunkMeta, bool, error) {
					// Simulate chunk not found, needs full processing
					return store.ChunkMeta{}, false, nil
				},
				UpsertChunkFunc: func(ctx context.Context, c models.Chunk, summaryVec []float32, contentHash string) error {
					// Validate the chunk data
					if c.Repository != "test/repo" {
						t.Errorf("Expected repository 'test/repo', got '%s'", c.Repository)
					}
					if c.Path != "main.go" {
						t.Errorf("Expected path 'main.go', got '%s'", c.Path)
					}
					if c.Language != "go" {
						t.Errorf("Expected language 'go', got '%s'", c.Language)
					}
					if c.LineStart != 1 || c.LineEnd != 5 {
						t.Errorf("Expected lines 1-5, got %d-%d", c.LineStart, c.LineEnd)
					}
					if c.Summary != "Go main package summary" {
						t.Errorf("Expected AI summary, got '%s'", c.Summary)
					}
					if !reflect.DeepEqual(summaryVec, []float32{0.5, 0.6, 0.7}) {
						t.Errorf("Expected embedding [0.5 0.6 0.7], got %v", summaryVec)
					}
					return nil
				},
			},
			mockClient: &MockAIClient{
				SummarizeFunc: func(ctx context.Context, filePath, language, content string) (string, error) {
					if filePath != "main.go" {
						t.Errorf("Expected filePath 'main.go', got '%s'", filePath)
					}
					if language != "go" {
						t.Errorf("Expected language 'go', got '%s'", language)
					}
					return "Go main package summary", nil
				},
				EmbedFunc: func(text string) ([]float32, error) {
					if text != "Go main package summary" {
						t.Errorf("Expected embedding text 'Go main package summary', got '%s'", text)
					}
					return []float32{0.5, 0.6, 0.7}, nil
				},
			},
			expectedError: nil,
		},
		{
			name:       "chunk already exists with same hash - skip processing",
			repoRoot:   "/test/repo",
			repository: "test/repo",
			files: map[string]string{
				"/test/repo/existing.py": "print('hello world')",
			},
			mockStore: &MockIndexableStore{
				GetChunkMetaFunc: func(ctx context.Context, repository, path string, ls, le int) (store.ChunkMeta, bool, error) {
					// Simulate chunk exists with same hash and summary
					expectedHash := hashContent("print('hello world')")
					return store.ChunkMeta{
						ContentHash:   expectedHash,
						Summary:       "Python print statement",
						HasSummaryVec: true,
					}, true, nil
				},
				UpsertChunkFunc: func(ctx context.Context, c models.Chunk, summaryVec []float32, contentHash string) error {
					// Should still call upsert but with existing summary and no new embedding
					if c.Summary != "Python print statement" {
						t.Errorf("Expected existing summary to be preserved")
					}
					if summaryVec != nil {
						t.Errorf("Expected no new embedding for existing chunk")
					}
					return nil
				},
			},
			mockClient: &MockAIClient{
				SummarizeFunc: func(ctx context.Context, filePath, language, content string) (string, error) {
					t.Error("Summarize should not be called for existing chunk with same hash")
					return "", nil
				},
				EmbedFunc: func(text string) ([]float32, error) {
					t.Error("Embed should not be called for existing chunk with same hash")
					return nil, nil
				},
			},
			expectedError: nil,
		},
		{
			name:       "AI summarization fails - use heuristic",
			repoRoot:   "/test/repo",
			repository: "test/repo",
			files: map[string]string{
				"/test/repo/script.sh": "#!/bin/bash\necho 'Hello from script'",
			},
			mockStore: &MockIndexableStore{
				GetChunkMetaFunc: func(ctx context.Context, repository, path string, ls, le int) (store.ChunkMeta, bool, error) {
					return store.ChunkMeta{}, false, nil
				},
				UpsertChunkFunc: func(ctx context.Context, c models.Chunk, summaryVec []float32, contentHash string) error {
					// Should use heuristic summary when AI fails
					expected := summarizeHeuristic("#!/bin/bash\necho 'Hello from script'")
					if c.Summary != expected {
						t.Errorf("Expected heuristic summary '%s', got '%s'", expected, c.Summary)
					}
					return nil
				},
			},
			mockClient: &MockAIClient{
				SummarizeFunc: func(ctx context.Context, filePath, language, content string) (string, error) {
					return "", errors.New("AI service unavailable")
				},
				EmbedFunc: func(text string) ([]float32, error) {
					return []float32{0.1, 0.2}, nil
				},
			},
			expectedError: nil,
		},
		{
			name:       "very large file - summarize only first 400k characters",
			repoRoot:   "/test/repo",
			repository: "test/repo",
			files: map[string]string{
				"/test/repo/large.txt": strings.Repeat("x", 500000), // 500k characters
			},
			mockStore: &MockIndexableStore{
				GetChunkMetaFunc: func(ctx context.Context, repository, path string, ls, le int) (store.ChunkMeta, bool, error) {
					return store.ChunkMeta{}, false, nil
				},
				UpsertChunkFunc: func(ctx context.Context, c models.Chunk, summaryVec []float32, contentHash string) error {
					return nil
				},
			},
			mockClient: &MockAIClient{
				SummarizeFunc: func(ctx context.Context, filePath, language, content string) (string, error) {
					if len(content) != 400000 {
						t.Errorf("Expected content length 400000 for large file, got %d", len(content))
					}
					return "Large file summary", nil
				},
				EmbedFunc: func(text string) ([]float32, error) {
					return []float32{0.1}, nil
				},
			},
			expectedError: nil,
		},
		{
			name:       "skip binary and hidden files",
			repoRoot:   "/test/repo",
			repository: "test/repo",
			files: map[string]string{
				"/test/repo/image.png":     "binary data",
				"/test/repo/.git/config":   "git config",
				"/test/repo/vendor/lib.go": "vendor code",
				"/test/repo/main.go":       "package main",
			},
			mockStore: &MockIndexableStore{
				UpsertChunkFunc: func(ctx context.Context, c models.Chunk, summaryVec []float32, contentHash string) error {
					// Should only process main.go
					if c.Path != "main.go" {
						t.Errorf("Only main.go should be processed, got '%s'", c.Path)
					}
					return nil
				},
			},
			mockClient: &MockAIClient{
				SummarizeFunc: func(ctx context.Context, filePath, language, content string) (string, error) {
					return "summary", nil
				},
				EmbedFunc: func(text string) ([]float32, error) {
					return []float32{0.1}, nil
				},
			},
			expectedError: nil,
		},
		{
			name:       "store upsert error",
			repoRoot:   "/test/repo",
			repository: "test/repo",
			files: map[string]string{
				"/test/repo/main.go": "package main",
			},
			mockStore: &MockIndexableStore{
				GetChunkMetaFunc: func(ctx context.Context, repository, path string, ls, le int) (store.ChunkMeta, bool, error) {
					return store.ChunkMeta{}, false, nil
				},
				UpsertChunkFunc: func(ctx context.Context, c models.Chunk, summaryVec []float32, contentHash string) error {
					return errors.New("database connection failed")
				},
			},
			mockClient: &MockAIClient{
				SummarizeFunc: func(ctx context.Context, filePath, language, content string) (string, error) {
					return "summary", nil
				},
				EmbedFunc: func(text string) ([]float32, error) {
					return []float32{0.1}, nil
				},
			},
			expectedError: nil, // Run continues despite upsert errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock file system
			walker := &MockFileSystemWalker{
				FilesToProcess: make([]string, 0),
			}
			fileReader := &MockFileReader{
				Files: tt.files,
			}

			// Set up walker to process our test files
			for path := range tt.files {
				walker.FilesToProcess = append(walker.FilesToProcess, path)
			}

			// Create indexer with mocked dependencies
			indexer := NewWithDependencies(
				tt.mockStore,
				tt.repoRoot,
				tt.repository,
				tt.mockClient,
				walker,
				fileReader,
			)

			// Run the indexer
			ctx := context.Background()
			err := indexer.Run(ctx)

			// Check error expectations
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("Expected error '%v', got nil", tt.expectedError)
				} else if err.Error() != tt.expectedError.Error() {
					t.Errorf("Expected error '%v', got '%v'", tt.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Run custom validation if provided
			if tt.validateResults != nil {
				tt.validateResults(t, tt.mockStore, tt.mockClient)
			}
		})
	}
}

func TestIndexer_UtilityFunctions(t *testing.T) {
	t.Run("hashContent", func(t *testing.T) {
		// Test that same content produces same hash
		content := "test content"
		hash1 := hashContent(content)
		hash2 := hashContent(content)
		if hash1 != hash2 {
			t.Errorf("Same content should produce same hash")
		}

		// Test that different content produces different hash
		hash3 := hashContent("different content")
		if hash1 == hash3 {
			t.Errorf("Different content should produce different hash")
		}

		// Test expected format (hex string)
		if len(hash1) != 40 { // SHA-1 hex is 40 characters
			t.Errorf("Expected 40-character hex string, got %d characters", len(hash1))
		}
	})

	t.Run("shouldSkip", func(t *testing.T) {
		tests := []struct {
			path     string
			expected bool
		}{
			{"/project/main.go", false},
			{"/project/vendor/lib.go", true},
			{"/project/.git/config", true},
			{"/project/.terraform/state", true},
			{"/project/image.png", true},
			{"/project/document.pdf", true},
			{"/project/app.exe", true},
			{"/project/go.mod", true},
			{"/project/go.sum", true},
			{"/project/README.md", false},
			{"/project/script.sh", false},
		}

		for _, tt := range tests {
			result := shouldSkip(tt.path)
			if result != tt.expected {
				t.Errorf("shouldSkip(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		}
	})

	t.Run("guessLang", func(t *testing.T) {
		tests := []struct {
			path     string
			expected string
		}{
			{"main.go", "go"},
			{"script.py", "python"},
			{"README.md", "markdown"},
			{"config.yaml", "yaml"},
			{"config.yml", "yaml"},
			{"package.json", "json"},
			{"script.sh", "shell"},
			{"app.js", "javascript"},
			{"app.ts", "typescript"},
			{"Main.java", "java"},
			{"app.rb", "ruby"},
			{"infra.tf", "terraform"},
			{"unknown.xyz", "xyz"},
		}

		for _, tt := range tests {
			result := guessLang(tt.path)
			if result != tt.expected {
				t.Errorf("guessLang(%s) = %s, expected %s", tt.path, result, tt.expected)
			}
		}
	})

	t.Run("naiveChunk", func(t *testing.T) {
		content := "line1\nline2\nline3"
		chunks := naiveChunk("/test/file.go", content)

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk, got %d", len(chunks))
		}

		chunk := chunks[0]
		if chunk.Content != content {
			t.Errorf("Expected content to be preserved")
		}
		if chunk.LineStart != 1 {
			t.Errorf("Expected LineStart=1, got %d", chunk.LineStart)
		}
		if chunk.LineEnd != 3 {
			t.Errorf("Expected LineEnd=3, got %d", chunk.LineEnd)
		}
	})

	t.Run("summarizeHeuristic", func(t *testing.T) {
		// Test short content
		short := "short content"
		result := summarizeHeuristic(short)
		if result != short {
			t.Errorf("Short content should be unchanged")
		}

		// Test long content gets truncated
		long := strings.Repeat("x", 300)
		result = summarizeHeuristic(long)
		if len(result) != 240 {
			t.Errorf("Expected 240 characters, got %d", len(result))
		}

		// Test whitespace trimming
		padded := "  content with spaces  "
		result = summarizeHeuristic(padded)
		if result != "content with spaces" {
			t.Errorf("Expected trimmed content, got '%s'", result)
		}
	})

	t.Run("chunkID", func(t *testing.T) {
		// Test same inputs produce same ID
		id1 := chunkID("path/file.go", 1, 10)
		id2 := chunkID("path/file.go", 1, 10)
		if id1 != id2 {
			t.Errorf("Same inputs should produce same ID")
		}

		// Test different inputs produce different IDs
		id3 := chunkID("path/file.go", 1, 11)
		if id1 == id3 {
			t.Errorf("Different inputs should produce different IDs")
		}

		// Test expected format (hex string)
		if len(id1) != 40 {
			t.Errorf("Expected 40-character hex string, got %d characters", len(id1))
		}
	})

	t.Run("rel", func(t *testing.T) {
		result := rel("/project/root", "/project/root/src/main.go")
		expected := "src/main.go"
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}

		// Test error case returns original path
		result = rel("/invalid", "/project/root/src/main.go")
		if result == "" {
			t.Errorf("Should return original path when rel fails")
		}
	})
}

func TestNewIndexer(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		store := &MockIndexableStore{}
		clientConfig := &ai.ClientConfig{
			Provider: ai.ProviderStub,
			Dim:      128,
		}

		indexer, err := New(store, "/test/repo", "test/repo", clientConfig)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if indexer == nil {
			t.Fatal("Expected indexer to be created")
		}
		if indexer.Store != store {
			t.Error("Store not set correctly")
		}
		if indexer.RepoRoot != "/test/repo" {
			t.Error("RepoRoot not set correctly")
		}
		if indexer.Repository != "test/repo" {
			t.Error("Repository not set correctly")
		}
	})

	t.Run("AI client creation failure", func(t *testing.T) {
		store := &MockIndexableStore{}
		clientConfig := &ai.ClientConfig{
			Provider: "invalid",
		}

		indexer, err := New(store, "/test/repo", "test/repo", clientConfig)

		if err == nil {
			t.Error("Expected error for invalid client config")
		}
		if indexer != nil {
			t.Error("Expected nil indexer on error")
		}
	})
}

func TestNewWithDependencies(t *testing.T) {
	store := &MockIndexableStore{}
	client := &MockAIClient{}
	walker := &MockFileSystemWalker{}
	fileReader := &MockFileReader{}

	indexer := NewWithDependencies(store, "/test", "test", client, walker, fileReader)

	if indexer.Store != store {
		t.Error("Store not set correctly")
	}
	if indexer.Client != client {
		t.Error("Client not set correctly")
	}
	if indexer.Walker != walker {
		t.Error("Walker not set correctly")
	}
	if indexer.FileReader != fileReader {
		t.Error("FileReader not set correctly")
	}
}

// Benchmark tests
func BenchmarkIndexer_HashContent(b *testing.B) {
	content := strings.Repeat("benchmark content ", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hashContent(content)
	}
}

func BenchmarkIndexer_SummarizeHeuristic(b *testing.B) {
	content := strings.Repeat("content for heuristic summary ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = summarizeHeuristic(content)
	}
}

func BenchmarkIndexer_ShouldSkip(b *testing.B) {
	paths := []string{
		"/project/main.go",
		"/project/vendor/lib.go",
		"/project/.git/config",
		"/project/image.png",
		"/project/script.sh",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			_ = shouldSkip(path)
		}
	}
}

// Test interface compliance
func TestInterfaceCompliance(t *testing.T) {
	var _ store.ChunkStore = &MockIndexableStore{}
	var _ FileSystemWalker = &MockFileSystemWalker{}
	var _ FileReader = &MockFileReader{}
	var _ ai.Client = &MockAIClient{}
}
