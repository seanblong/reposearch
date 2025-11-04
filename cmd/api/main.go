package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/seanblong/reposearch/internal/ai"
	"github.com/seanblong/reposearch/internal/auth"
	"github.com/seanblong/reposearch/internal/config"
	"github.com/seanblong/reposearch/internal/search"
	"github.com/seanblong/reposearch/internal/store"
	"github.com/seanblong/reposearch/pkg/models"
	"github.com/spf13/pflag"
)

type Simple struct {
	Path       string  `json:"path"`
	Language   string  `json:"language"`
	LineStart  int     `json:"line_start"`
	LineEnd    int     `json:"line_end"`
	Score      float64 `json:"score"`
	Preview    string  `json:"preview"`
	Summary    string  `json:"summary,omitempty"`
	Ref        string  `json:"ref,omitempty"`
	Repository string  `json:"repository,omitempty"`
}

func output(res []models.SearchResult) (out []Simple) {
	out = make([]Simple, 0, len(res))
	for _, r := range res {
		score := r.Score
		if math.IsNaN(score) || math.IsInf(score, 0) {
			score = 0
		}
		// Build a small preview (first 400 chars)
		preview := r.Chunk.Content
		// if len(preview) > 400 {
		// 	preview = preview[:400] + "…"
		// }
		out = append(out, Simple{
			Path:       r.Chunk.Path,
			Language:   r.Chunk.Language,
			LineStart:  r.Chunk.LineStart,
			LineEnd:    r.Chunk.LineEnd,
			Score:      score,
			Preview:    preview,
			Summary:    r.Chunk.Summary,
			Ref:        r.Chunk.Ref,
			Repository: r.Chunk.Repository,
		})
	}
	return out
}

func main() {
	// Create flagset for configuration
	fs := pflag.NewFlagSet("reposearch-api", pflag.ExitOnError)

	// Load configuration
	cfg, err := config.Load("", fs)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	fs.Usage = cfg.Usage

	// Set up logging
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Invalid log level '%s': %v", cfg.LogLevel, err)
	}
	logger := zerolog.New(os.Stdout).Level(level).With().Timestamp().Logger()
	logger.Info().Str("provider", cfg.Provider).Str("log_level", cfg.LogLevel).Bool("auth_enabled", cfg.Auth.Enabled).Msg("starting reposearch api")

	// Create AI client configuration
	var clientConfig *ai.ClientConfig
	switch strings.ToLower(cfg.Provider) {
	case "openai":
		clientConfig = &ai.ClientConfig{
			APIKey:       cfg.APIKey,
			EmbedModel:   cfg.EmbedModel,
			SummaryModel: cfg.SummaryModel,
			Dim:          cfg.Dim,
			ProjectID:    cfg.ProjectID,
			Provider:     ai.ProviderOpenAI,
		}
	case "vertexai", "google":
		clientConfig = &ai.ClientConfig{
			APIKey:       cfg.APIKey,
			EmbedModel:   cfg.EmbedModel,
			SummaryModel: cfg.SummaryModel,
			Dim:          cfg.Dim,
			ProjectID:    cfg.ProjectID,
			Location:     cfg.Location,
			Provider:     ai.ProviderVertexAI,
		}
	case "stub":
		clientConfig = &ai.ClientConfig{
			Dim:      cfg.Dim,
			Provider: ai.ProviderStub,
		}
	default:
		log.Fatalf("unsupported provider: %s", cfg.Provider)
	}

	// Initialize auth with configuration
	auth.InitializeAuth(
		cfg.Auth.JwtSecret,
		cfg.Auth.GithubClientID,
		cfg.Auth.GithubClientSecret,
		cfg.Auth.GithubRedirectURL,
		cfg.Auth.GithubAllowedOrg,
		cfg.Auth.Enabled,
	)

	ctx := context.Background()
	st, err := store.New(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer st.Close()

	c, err := ai.NewClient(clientConfig)
	if err != nil {
		log.Fatalf("Failed to create AI client: %v", err)
	}

	// Use the AI client's dimension for database migration
	dim := c.Dim()
	logger.Info().Int("embedding_dim", dim).Str("embed_model", clientConfig.EmbedModel).Msg("AI client initialized")

	if err := st.Migrate(ctx, dim); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	svc := search.NewService(c, st)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

	// Auth status endpoint (always available)
	mux.HandleFunc("/auth/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]bool{"enabled": auth.IsAuthEnabled()})
		if err != nil {
			http.Error(w, "Failed to encode response", 500)
		}
	})

	// Authentication endpoints (only if auth is enabled)
	if auth.IsAuthEnabled() {
		log.Println("Authentication is ENABLED")

		mux.HandleFunc("/auth/github", func(w http.ResponseWriter, r *http.Request) {
			state := auth.GenerateState()

			// Store state in cookie for validation
			http.SetCookie(w, &http.Cookie{
				Name:     "oauth_state",
				Value:    state,
				Path:     "/",
				MaxAge:   600, // 10 minutes
				HttpOnly: true,
				Secure:   strings.HasPrefix(r.Header.Get("X-Forwarded-Proto"), "https"),
				SameSite: http.SameSiteLaxMode,
			})

			loginURL := auth.GetGithubLoginURL(state)
			http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
		})

		mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			state := r.URL.Query().Get("state")

			// Validate state
			stateCookie, err := r.Cookie("oauth_state")
			if err != nil || stateCookie.Value != state {
				http.Error(w, "Invalid state parameter", http.StatusBadRequest)
				return
			}

			// Clear state cookie
			http.SetCookie(w, &http.Cookie{
				Name:   "oauth_state",
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})

			if code == "" {
				http.Error(w, "Missing code parameter", http.StatusBadRequest)
				return
			}

			// Exchange code for token
			accessToken, err := auth.ExchangeCodeForToken(code)
			if err != nil {
				http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
				return
			}

			// Get user info
			user, err := auth.GetGithubUser(accessToken)
			if err != nil {
				http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Generate JWT
			token, err := auth.GenerateJWT(user)
			if err != nil {
				http.Error(w, "Failed to generate token", http.StatusInternalServerError)
				return
			}

			// Set cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "auth_token",
				Value:    token,
				Path:     "/",
				MaxAge:   86400, // 24 hours
				HttpOnly: true,
				Secure:   strings.HasPrefix(r.Header.Get("X-Forwarded-Proto"), "https"),
				SameSite: http.SameSiteLaxMode,
			})

			// Return user info and token
			w.Header().Set("Content-Type", "application/json")
			err = json.NewEncoder(w).Encode(auth.AuthResponse{
				User:  *user,
				Token: token,
			})
			if err != nil {
				http.Error(w, "Failed to encode response", 500)
			}
		})

		mux.HandleFunc("/auth/me", func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header or cookie
			var tokenString string

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				if cookie, err := r.Cookie("auth_token"); err == nil {
					tokenString = cookie.Value
				}
			}

			if tokenString == "" {
				http.Error(w, "No authentication token", http.StatusUnauthorized)
				return
			}

			user, err := auth.ValidateJWT(tokenString)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			err = json.NewEncoder(w).Encode(auth.AuthResponse{
				User:  *user,
				Token: tokenString,
			})
			if err != nil {
				http.Error(w, "Failed to encode response", 500)
			}
		})

		mux.HandleFunc("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			// Clear cookie
			http.SetCookie(w, &http.Cookie{
				Name:   "auth_token",
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})

			w.WriteHeader(http.StatusOK)
		})
	} else {
		log.Println("Authentication is DISABLED - running in open mode")
	}

	mux.HandleFunc("/repositories", auth.OptionalAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		repos, err := st.GetRepositories(ctx)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(repos); err != nil {
			http.Error(w, "Failed to encode repositories", 500)
		}
	}))
	mux.HandleFunc("/repositories/", auth.OptionalAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Support paths like /repositories/{repo}/refs where {repo} may contain '/'
		// e.g. repo encoded as owner%2Frepo by the frontend.
		rel := strings.TrimPrefix(r.URL.Path, "/repositories/")
		// Normalize trailing slash
		rel = strings.TrimSuffix(rel, "/")

		// If URL ends with /refs, extract the repo portion before it.
		if strings.HasSuffix(rel, "/refs") {
			repoPart := strings.TrimSuffix(rel, "/refs")
			// Remove any leading slash remaining
			repoPart = strings.TrimPrefix(repoPart, "/")
			repoName, err := url.PathUnescape(repoPart)
			if err != nil {
				http.Error(w, "Invalid repository path", http.StatusBadRequest)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			refs, err := st.GetRefs(ctx, repoName)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(refs); err != nil {
				http.Error(w, "Failed to encode refs", 500)
			}
			return
		}

		http.NotFound(w, r)
	}))
	mux.HandleFunc("/search", auth.OptionalAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		q := r.URL.Query().Get("q")
		k := 5
		if v := r.URL.Query().Get("k"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				k = n
			}
		}
		if q == "" {
			http.Error(w, "missing query parameter q", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		opt := store.QueryOpts{
			Language:     r.URL.Query().Get("language"), // e.g. "shell"
			PathContains: r.URL.Query().Get("path_contains"),
			Repository:   r.URL.Query().Get("repository"),
			Ref:          r.URL.Query().Get("ref"),
		}
		res, err := svc.Query(ctx, q, k, opt)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// original full payload (but never empty body)
		w.Header().Set("Content-Type", "application/json")
		if res == nil {
			if _, err := w.Write([]byte("[]")); err != nil {
				http.Error(w, "Failed to write response", http.StatusInternalServerError)
				return
			}
		} else {
			for i := range res {
				if math.IsNaN(res[i].Score) || math.IsInf(res[i].Score, 0) {
					res[i].Score = 0
				}
			}
			if err := json.NewEncoder(w).Encode(res); err != nil {
				log.Printf("failed to encode response: %v", err)
				// fallback to an empty JSON array if encoding or writing fails
				_, _ = w.Write([]byte("[]"))
			}
		}

		hlog.FromRequest(r).Info().Str("path", "/search").Str("q", q).Int("k", k).Dur("dur", time.Since(start)).Msg("served")
	}))

	handler := hlog.NewHandler(logger)(
		hlog.AccessHandler(func(r *http.Request, status, size int, dur time.Duration) {
			logger.Info().Str("method", r.Method).Str("path", r.URL.Path).Int("status", status).Int("size", size).Dur("dur", dur).Msg("http")
		})(mux),
	)

	address := fmt.Sprintf(":%d", cfg.Port)
	s := &http.Server{Addr: address, Handler: handler}
	logger.Info().Str("addr", s.Addr).Msg("api server listening")
	log.Fatal(s.ListenAndServe())
}
