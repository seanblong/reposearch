package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/seanblong/reposearch/internal/ai"
	"github.com/seanblong/reposearch/internal/config"
	"github.com/seanblong/reposearch/internal/indexer"
	"github.com/seanblong/reposearch/internal/store"
	"github.com/spf13/pflag"
)

func main() {
	fs := pflag.NewFlagSet("reposearch-api", pflag.ExitOnError)

	cfg, err := config.Load("", fs)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	fs.Usage = cfg.Usage

	repo := cfg.RepoRoot
	if cfg.RepoURL != "" {
		var err error
		repo, err = cloneToTemp(cfg.RepoURL, cfg.GitRef, cfg.GithubToken)
		if err != nil {
			log.Fatalf("clone failed: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(repo); err != nil {
				log.Printf("Failed to remove temp directory %s: %v", repo, err)
			}
		}()
	} else {
		cfg.RepoURL = "local"
	}

	provider := strings.ToLower(cfg.Provider)
	log.Printf("using provider: %s", provider)
	var clientConfig *ai.ClientConfig
	switch provider {
	case "openai":
		clientConfig = &ai.ClientConfig{
			APIKey:       cfg.APIKey,
			EmbedModel:   cfg.EmbedModel,
			SummaryModel: cfg.SummaryModel,
			Dim:          cfg.Dim,
			ProjectID:    cfg.ProjectID,
			Provider:     ai.ProviderOpenAI,
		}
	case "vertexai":
		clientConfig = &ai.ClientConfig{
			APIKey:       cfg.APIKey,
			EmbedModel:   cfg.EmbedModel,
			SummaryModel: cfg.SummaryModel,
			Dim:          cfg.Dim,
			ProjectID:    cfg.ProjectID,
			Provider:     ai.ProviderVertexAI,
		}
	case "stub":
		clientConfig = &ai.ClientConfig{
			Dim:      cfg.Dim,
			Provider: ai.ProviderStub,
		}
	default:
		log.Fatalf("unsupported provider: %s", provider)
	}

	ctx := context.Background()

	// Initialize store
	st, err := store.New(ctx, cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	ix, err := indexer.New(st, repo, cfg.RepoURL, clientConfig)
	if err != nil {
		log.Fatal(err)
	}

	// if pulling in a local directory set ref to directory name
	if cfg.RepoURL == "local" {
		parts := strings.Split(strings.TrimRight(repo, "/"), string(os.PathSeparator))
		ix.Ref = parts[len(parts)-1]
	} else {
		ix.Ref = cfg.GitRef
	}

	if ix.Client.Dim() == 0 {
		log.Fatal("embedding dimension must be set")
	}

	if err := st.Migrate(ctx, ix.Client.Dim()); err != nil {
		log.Fatal(err)
	}

	if err := ix.Run(ctx); err != nil {
		log.Fatal(err)
	}
}

func cloneToTemp(repoURL, ref, token string) (string, error) {
	dir, err := os.MkdirTemp("", "reposearch-*")
	if err != nil {
		return "", err
	}
	url := repoURL
	if token != "" && strings.HasPrefix(url, "https://") {
		url = "https://" + token + ":x-oauth-basic@" + strings.TrimPrefix(url, "https://")
	}
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", ref, url, dir)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			log.Printf("Failed to remove temp directory %s: %v", dir, rmErr)
		}
		return "", fmt.Errorf("git clone: %w", err)
	}
	return dir, nil
}
