package indexer

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/karrick/godirwalk"
	"github.com/rs/zerolog/log"
	"github.com/seanblong/reposearch/internal/ai"
	"github.com/seanblong/reposearch/internal/store"
	"github.com/seanblong/reposearch/pkg/models"
)

// FileSystemWalker defines the interface for walking directories
type FileSystemWalker interface {
	Walk(root string, options *godirwalk.Options) error
}

// FileReader defines the interface for reading files
type FileReader interface {
	ReadFile(filename string) ([]byte, error)
}

// DefaultFileSystemWalker implements FileSystemWalker using godirwalk
type DefaultFileSystemWalker struct{}

func (d *DefaultFileSystemWalker) Walk(root string, options *godirwalk.Options) error {
	return godirwalk.Walk(root, options)
}

// DefaultFileReader implements FileReader using os
type DefaultFileReader struct{}

func (d *DefaultFileReader) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// Indexer handles indexing of a code repository.
type Indexer struct {
	Store      store.ChunkStore
	RepoRoot   string
	Repository string
	Ref        string
	Client     ai.Client
	Walker     FileSystemWalker
	FileReader FileReader
}

// hashContent returns the SHA-1 hash of the given content as a hex string.
func hashContent(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// New creates a new Indexer instance.
func New(s store.ChunkStore, repoRoot string, repository string, clientConfig *ai.ClientConfig) (*Indexer, error) {
	client, err := ai.NewClient(clientConfig)
	if err != nil {
		return nil, err
	}

	return &Indexer{
		Store:      s,
		RepoRoot:   repoRoot,
		Repository: repository,
		Client:     client,
		Walker:     &DefaultFileSystemWalker{},
		FileReader: &DefaultFileReader{},
	}, nil
}

// NewWithDependencies creates a new Indexer instance with custom dependencies for testing
func NewWithDependencies(store store.ChunkStore, repoRoot string, repository string, client ai.Client, walker FileSystemWalker, fileReader FileReader) *Indexer {
	return &Indexer{
		Store:      store,
		RepoRoot:   repoRoot,
		Repository: repository,
		Client:     client,
		Walker:     walker,
		FileReader: fileReader,
	}
}

// workItem represents a file to be processed
type workItem struct {
	path    string
	content string
}

// processWorkItem handles the processing of a single file
func (ix *Indexer) processWorkItem(ctx context.Context, item workItem) error {
	chunks := naiveChunk(item.path, item.content)
	for _, ch := range chunks {
		relPath := rel(ix.RepoRoot, item.path)
		lang := guessLang(item.path)
		hash := hashContent(ch.Content)

		var needSummary, needEmbed bool

		meta, found, err := ix.Store.GetChunkMeta(ctx, ix.Repository, relPath, ch.LineStart, ch.LineEnd)
		if err != nil {
			// If there's an error getting metadata, we need both summary and embedding
			needSummary = true
			needEmbed = true
		} else {
			// Decide what we need based on existing metadata
			needSummary = !found || meta.ContentHash != hash || meta.Summary == ""
			needEmbed = !found || meta.ContentHash != hash || !meta.HasSummaryVec
		}

		var summary string
		if needSummary {
			if ix.Client != nil {
				// if content is long, we can just summarize the start
				if len(ch.Content) > 400_000 {
					if s, err := ix.Client.Summarize(ctx, relPath, lang, ch.Content[:400_000]); err == nil && strings.TrimSpace(s) != "" {
						summary = s
					} else {
						log.Warn().Err(err).Str("path", item.path).Msg("summarization failed, using heuristic")
						summary = summarizeHeuristic(ch.Content)
					}
				} else {
					if s, err := ix.Client.Summarize(ctx, relPath, lang, ch.Content); err == nil && strings.TrimSpace(s) != "" {
						summary = s
					} else {
						log.Warn().Err(err).Str("path", item.path).Msg("summarization failed, using heuristic")
						summary = summarizeHeuristic(ch.Content)
					}
				}
			} else {
				log.Warn().Str("path", item.path).Msg("no summarizer client, using heuristic")
				summary = summarizeHeuristic(ch.Content)
			}
		} else {
			// Use existing summary if we don't need a new one
			summary = meta.Summary
		}

		id := chunkID(relPath, ch.LineStart, ch.LineEnd)
		var summaryVec []float32 // Only embed the summary
		if needEmbed {
			summaryVec, _ = ix.Client.Embed(summary)
		}
		m := models.Chunk{
			ID: id, Repository: ix.Repository, Ref: ix.Ref, Path: relPath, Language: lang,
			Summary: summary, Content: ch.Content,
			LineStart: ch.LineStart, LineEnd: ch.LineEnd,
		}
		log.Info().Str("path", relPath).
			Int("lines", ch.LineEnd-ch.LineStart+1).
			Bool("need_summary", needSummary).
			Bool("need_embed", needEmbed).
			Msg("indexing chunk")
		if err := ix.Store.UpsertChunk(ctx, m, summaryVec, hash); err != nil {
			log.Error().Err(err).Str("path", item.path).Msg("upsert failed")
		}
	}
	return nil
}

func (ix *Indexer) Run(ctx context.Context) error {
	// Determine number of workers (default to number of CPU cores)
	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8 // Cap at 8 to avoid overwhelming the AI API
	}

	log.Info().Int("workers", numWorkers).Msg("starting concurrent indexing")

	// Create channels for work distribution
	workChan := make(chan workItem, numWorkers*2) // Buffer to keep workers busy
	errorChan := make(chan error, 1)

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			log.Debug().Int("worker", workerID).Msg("worker started")

			for item := range workChan {
				if err := ix.processWorkItem(ctx, item); err != nil {
					select {
					case errorChan <- err:
					default:
						// Error channel is full, log the error
						log.Error().Err(err).Str("path", item.path).Msg("worker processing error")
					}
				}
			}

			log.Debug().Int("worker", workerID).Msg("worker finished")
		}(i)
	}

	// Start a goroutine to close errorChan when all workers are done
	go func() {
		wg.Wait()
		close(errorChan)
	}()

	// Walk files and send them to workers
	walkErr := ix.Walker.Walk(ix.RepoRoot, &godirwalk.Options{
		Unsorted: true,
		Callback: func(path string, de *godirwalk.Dirent) error {
			// Handle test case where de might be nil (for MockFileSystemWalker)
			if de != nil && de.IsDir() {
				return nil
			}
			if shouldSkip(path) {
				return nil
			}

			b, err := ix.FileReader.ReadFile(path)
			if err != nil {
				log.Warn().Err(err).Str("path", path).Msg("failed to read file")
				return nil
			}

			// Send work item to channel
			select {
			case workChan <- workItem{path: path, content: string(b)}:
				// Successfully sent to worker
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		},
	})

	// Close work channel to signal workers to finish
	close(workChan)

	// Wait for all workers to complete
	wg.Wait()

	// Check for any errors
	select {
	case err := <-errorChan:
		if err != nil {
			return err
		}
	default:
	}

	return walkErr
}

// chunk holds a piece of a file.
type chunk struct {
	Content            string
	LineStart, LineEnd int
}

// naiveChunk splits the content into a single chunk.
func naiveChunk(path, content string) []chunk {
	lines := strings.Count(content, "\n") + 1
	return []chunk{{Content: content, LineStart: 1, LineEnd: lines}}
}

// summarizeHeuristic provides a simple heuristic summary by truncating the content.
func summarizeHeuristic(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 240 {
		s = s[:240]
	}
	return s
}

// shouldSkip returns true if the file at path should be skipped.
func shouldSkip(path string) bool {
	p := strings.ToLower(path)
	if strings.Contains(p, "/vendor/") ||
		strings.Contains(p, "/.git/") ||
		strings.Contains(p, "/.terraform/") ||
		strings.Contains(p, "/node_modules/") ||
		strings.Contains(p, "/target/") ||
		strings.Contains(p, "/build/") ||
		strings.Contains(p, "/dist/") ||
		strings.Contains(p, "/out/") ||
		strings.Contains(p, "/bin/") ||
		strings.Contains(p, "/obj/") ||
		strings.Contains(p, "/.venv/") ||
		strings.Contains(p, "/venv/") ||
		strings.Contains(p, "/__pycache__/") ||
		strings.Contains(p, "/.pytest_cache/") ||
		strings.Contains(p, "/.gradle/") ||
		strings.Contains(p, "/.m2/") ||
		strings.Contains(p, "/.idea/") ||
		strings.Contains(p, "/coverage/") ||
		strings.Contains(p, "/.cache/") {
		return true
	}
	switch filepath.Ext(p) {
	case ".png", ".jpg", ".jpeg", ".gif", ".pdf", ".webp", ".lock", ".zip", ".svg", ".exe", ".dll", ".xml", ".sum", ".mod", ".sql":
		return true
	}
	return false
}

func rel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return r
}

func chunkID(path string, a, b int) string {
	h := sha1.Sum([]byte(path + "#" + fmtI(a) + ":" + fmtI(b)))
	return hex.EncodeToString(h[:])
}

func fmtI(i int) string { return fmt.Sprintf("%d", i) }

func guessLang(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".sh":
		return "shell"
	case ".py":
		return "python"
	case ".go":
		return "go"
	case ".md":
		return "markdown"
	case ".tf":
		return "terraform"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	default:
		return strings.TrimPrefix(ext, ".")
	}
}
