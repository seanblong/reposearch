package search

import (
	"context"
	"log"
	"strings"

	"github.com/seanblong/reposearch/internal/ai"
	"github.com/seanblong/reposearch/internal/store"
	"github.com/seanblong/reposearch/pkg/models"
)

type Service struct {
	Client ai.Client
	Store  store.ChunkStore
}

// NewService creates a new search service with the provided AI client and store
func NewService(client ai.Client, store store.ChunkStore) *Service {
	return &Service{
		Client: client,
		Store:  store,
	}
}

func (s *Service) Query(ctx context.Context, q string, k int, opt store.QueryOpts) ([]models.SearchResult, error) {
	q = strings.TrimSpace(q)
	opt.QueryText = q

	head, err := s.Client.Embed(q)
	if err != nil {
		log.Printf("AI CLIENT ERROR: Embedding failed for query '%s': %v", q, err)
		log.Printf("This likely indicates AI authentication issues (e.g., missing 'gcloud auth login' for Vertex AI, invalid API key, etc.)")
		log.Printf("Proceeding with empty embedding vector - search results may be poor or empty")
		head = nil
	}

	res, err := s.Store.Search(ctx, head, k, opt)
	if err != nil {
		return nil, err
	}
	return res, nil
}
