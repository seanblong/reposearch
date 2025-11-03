package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type Specification struct {
	Provider     string            `yaml:"provider"`
	APIKey       string            `yaml:"providerApiKey" envconfig:"PROVIDER_API_KEY"`
	EmbedModel   string            `yaml:"providerEmbedModel" envconfig:"PROVIDER_EMBEDDING_MODEL"`
	SummaryModel string            `yaml:"providerSummaryModel" envconfig:"PROVIDER_SUMMARY_MODEL"`
	ProjectID    string            `yaml:"providerProjectID" envconfig:"PROVIDER_PROJECT_ID"`
	Location     string            `yaml:"providerLocation" envconfig:"PROVIDER_LOCATION"`
	Dim          int               `yaml:"providerDim" envconfig:"EMBED_DIM"`
	Database     string            `yaml:"database" envconfig:"DB_URL"`
	RepoRoot     string            `yaml:"repoRoot" split_words:"true"`
	RepoURL      string            `yaml:"repoURL" split_words:"true"`
	GithubToken  string            `yaml:"githubToken" envconfig:"GITHUB_TOKEN"`
	GitRef       string            `yaml:"gitRef" split_words:"true"`
	LogLevel     string            `yaml:"logLevel" split_words:"true"`
	Port         int               `yaml:"port" split_words:"true"`
	Auth         AuthSpecification `yaml:"auth"`

	flags *pflag.FlagSet `ignored:"true"`
}

type AuthSpecification struct {
	Enabled            bool   `yaml:"enabled"`
	JwtSecret          string `yaml:"jwtSecret" split_words:"true"`
	GithubClientID     string `yaml:"githubClientID" split_words:"true"`
	GithubClientSecret string `yaml:"githubClientSecret" split_words:"true"`
	GithubRedirectURL  string `yaml:"githubRedirectURL" split_words:"true"`
	GithubAllowedOrg   string `yaml:"githubAllowedOrg" split_words:"true"`
}

const envPrefix = "REPOSEARCH"

func (s *Specification) Usage() {
	fmt.Fprint(os.Stderr, s.flags.FlagUsages())
}

// Load => defaults < YAML < env < flags.
// configPath may be ""; if so we auto-discover.
func Load(configPath string, fs *pflag.FlagSet) (Specification, error) {
	var cfg Specification

	// set defaults (lowest precedence)
	setDefaults(&cfg)
	bindFlags(fs, &cfg)

	// config file
	path := configPath
	if path == "" {
		if v := os.Getenv(envPrefix + "_CONFIG"); v != "" {
			path = v
		} else {
			for _, cand := range []string{
				"config/reposearch.yaml",
				"config/config.yaml",
				"./reposearch.yaml",
				"./config.yaml",
			} {
				if fileExists(cand) {
					path = cand
					break
				}
			}
		}
	}

	if path != "" {
		if !fileExists(path) {
			return Specification{}, fmt.Errorf("config file not found: %s", path)
		}
		if err := loadYAML(path, &cfg); err != nil {
			return Specification{}, fmt.Errorf("load yaml %s: %w", path, err)
		}

	}

	// env overrides config file
	if err := envconfig.Process(envPrefix, &cfg); err != nil {
		return Specification{}, fmt.Errorf("env override: %w", err)
	}

	// flags override everything
	if err := fs.Parse(os.Args[1:]); err != nil {
		return Specification{}, err
	}
	applyChangedFlags(fs, &cfg)

	// Minimal sanity
	if strings.TrimSpace(cfg.Database) == "" {
		return Specification{}, fmt.Errorf("REPOSEARCH_DB_URL is required (env/file/flag)")
	}
	if strings.TrimSpace(cfg.LogLevel) == "" {
		cfg.LogLevel = "info"
	}
	return cfg, nil
}

// ---------- helpers ----------

func loadYAML(path string, into any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, into)
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func bindFlags(fs *pflag.FlagSet, c *Specification) {
	fs.String("config", "", "Path to config file")

	// If --config is provided on the command line, capture it now so
	// config discovery (which runs before flags.Parse) can use it.
	for i, a := range os.Args {
		if a == "--config" {
			if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
				_ = os.Setenv(envPrefix+"_CONFIG", os.Args[i+1])
			}
		} else if strings.HasPrefix(a, "--config=") {
			parts := strings.SplitN(a, "=", 2)
			if len(parts) == 2 {
				_ = os.Setenv(envPrefix+"_CONFIG", parts[1])
			}
		}
	}

	fs.String("provider", c.Provider, "Provider (e.g., stub, openai, google)")
	fs.String("provider-api-key", c.APIKey, "Provider API key")
	fs.String("provider-embedding-model", c.EmbedModel, "Provider embedding model")
	fs.String("provider-summary-model", c.SummaryModel, "Provider summary model")
	fs.String("provider-project-id", c.ProjectID, "Provider project ID")
	fs.String("provider-location", c.Location, "Provider location/region")

	fs.Int("embed-dim", c.Dim, "Embedding dimensionality")

	fs.String("db-url", c.Database, "Database URL (DSN)")

	fs.String("repo-root", c.RepoRoot, "Path to local repo root")
	fs.String("git-repo", c.RepoURL, "Git repository URL")
	fs.String("github-token", c.GithubToken, "GitHub API token")
	fs.String("git-ref", c.GitRef, "Git reference (branch/tag/sha)")

	fs.String("log-level", c.LogLevel, "Log level (debug|info|warn|error)")
	fs.Int("port", c.Port, "API server port")

	fs.Bool("auth-enabled", c.Auth.Enabled, "Enable GitHub OAuth authentication")
	fs.String("auth-jwt-secret", c.Auth.JwtSecret, "JWT secret for signing tokens")
	fs.String("auth-github-client-id", c.Auth.GithubClientID, "GitHub OAuth App Client ID")
	fs.String("auth-github-client-secret", c.Auth.GithubClientSecret, "GitHub OAuth App Client Secret")
	fs.String("auth-github-redirect-url", c.Auth.GithubRedirectURL, "GitHub OAuth App Redirect URL")
	fs.String("auth-github-allowed-org", c.Auth.GithubAllowedOrg, "Optional: Restrict login to a GitHub organization")

	// Used later for usage/help
	// create a shallow copy of fs (so Usage can be called safely without mutating caller)
	copied := pflag.NewFlagSet("temp", pflag.ContinueOnError)
	*copied = *fs
	c.flags = copied
}

func applyChangedFlags(fs *pflag.FlagSet, c *Specification) {
	setStr := func(name string, dst *string) {
		if fs.Changed(name) {
			v, _ := fs.GetString(name)
			*dst = v
		}
	}
	setInt := func(name string, dst *int) {
		if fs.Changed(name) {
			v, _ := fs.GetInt(name)
			*dst = v
		}
	}
	setBool := func(name string, dst *bool) {
		if fs.Changed(name) {
			v, _ := fs.GetBool(name)
			*dst = v
		}
	}

	// (We ignore --config here; it's for discovery.)
	setStr("provider", &c.Provider)
	setStr("provider-api-key", &c.APIKey)
	setStr("provider-embedding-model", &c.EmbedModel)
	setStr("provider-summary-model", &c.SummaryModel)
	setStr("provider-project-id", &c.ProjectID)
	setStr("provider-location", &c.Location)

	setInt("embed-dim", &c.Dim)

	setStr("db-url", &c.Database)

	setStr("repo-root", &c.RepoRoot)
	setStr("git-repo", &c.RepoURL)
	setStr("github-token", &c.GithubToken)
	setStr("git-ref", &c.GitRef)

	setStr("log-level", &c.LogLevel)
	setInt("port", &c.Port)

	// Auth flags
	setBool("auth-enabled", &c.Auth.Enabled)
	setStr("auth-jwt-secret", &c.Auth.JwtSecret)
	setStr("auth-github-client-id", &c.Auth.GithubClientID)
	setStr("auth-github-client-secret", &c.Auth.GithubClientSecret)
	setStr("auth-github-redirect-url", &c.Auth.GithubRedirectURL)
	setStr("auth-github-allowed-org", &c.Auth.GithubAllowedOrg)
}

func setDefaults(c *Specification) {
	c.LogLevel = "info"
	c.RepoRoot = "."
	c.GitRef = "main"
	c.GithubToken = ""
	c.Provider = "stub"
	c.Database = "postgres://postgres:postgres@localhost:5432/intent?sslmode=disable"
	c.Auth.GithubRedirectURL = "http://localhost:3000/auth/callback"
	c.Auth.Enabled = false
	c.Dim = 0
	c.Location = "us-central1"
	c.Port = 8080
}
