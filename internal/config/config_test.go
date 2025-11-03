package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestSpecificationDefaults(t *testing.T) {
	// Test that default values are properly set
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	// Clear any existing environment variables that might interfere
	clearTestEnv(t)

	cfg, err := Load("", fs)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Test default values
	expected := Specification{
		Provider: "stub",
		Location: "us-central1",
		Database: "postgres://postgres:postgres@localhost:5432/intent?sslmode=disable",
		RepoRoot: ".",
		GitRef:   "main",
		LogLevel: "info",
		Auth: AuthSpecification{
			Enabled:           false,
			GithubRedirectURL: "http://localhost:3000/auth/callback",
		},
	}

	if cfg.Provider != expected.Provider {
		t.Errorf("Expected Provider %q, got %q", expected.Provider, cfg.Provider)
	}
	if cfg.Location != expected.Location {
		t.Errorf("Expected Location %q, got %q", expected.Location, cfg.Location)
	}
	if cfg.Database != expected.Database {
		t.Errorf("Expected Database %q, got %q", expected.Database, cfg.Database)
	}
	if cfg.RepoRoot != expected.RepoRoot {
		t.Errorf("Expected RepoRoot %q, got %q", expected.RepoRoot, cfg.RepoRoot)
	}
	if cfg.GitRef != expected.GitRef {
		t.Errorf("Expected GitRef %q, got %q", expected.GitRef, cfg.GitRef)
	}
	if cfg.LogLevel != expected.LogLevel {
		t.Errorf("Expected LogLevel %q, got %q", expected.LogLevel, cfg.LogLevel)
	}
	if cfg.Auth.Enabled != expected.Auth.Enabled {
		t.Errorf("Expected Auth.Enabled %v, got %v", expected.Auth.Enabled, cfg.Auth.Enabled)
	}
	if cfg.Auth.JwtSecret != expected.Auth.JwtSecret {
		t.Errorf("Expected Auth.JwtSecret %q, got %q", expected.Auth.JwtSecret, cfg.Auth.JwtSecret)
	}
	if cfg.Auth.GithubRedirectURL != expected.Auth.GithubRedirectURL {
		t.Errorf("Expected Auth.GithubRedirectURL %q, got %q", expected.Auth.GithubRedirectURL, cfg.Auth.GithubRedirectURL)
	}
}

func TestLoadFromYAMLFile(t *testing.T) {
	// Create a temporary YAML file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")

	yamlContent := `
provider: "openai"
providerApiKey: "test-api-key"
providerEmbedModel: "text-embedding-3-small"
providerSummaryModel: "gpt-4o-mini"
providerProjectID: "test-project"
providerLocation: "us-west1"
providerDim: 1536
providerDatabase: "postgres://test:test@localhost:5432/testdb"
repoRoot: "/tmp/repo"
repoURL: "https://github.com/test/repo.git"
githubToken: "ghp_test123"
gitRef: "develop"
logLevel: "debug"
auth:
  enabled: true
  jwtSecret: "super-secret-key"
  githubClientID: "test-client-id"
  githubClientSecret: "test-client-secret"
  githubRedirectURL: "https://example.com/auth/callback"
  githubAllowedOrg: "test-org"
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Clear test environment but preserve the database URL that's in our YAML
	clearTestEnv(t)
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	cfg, err := Load(configFile, fs)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify YAML values were loaded
	if cfg.Provider != "openai" {
		t.Errorf("Expected Provider 'openai', got %q", cfg.Provider)
	}
	if cfg.APIKey != "test-api-key" {
		t.Errorf("Expected APIKey 'test-api-key', got %q", cfg.APIKey)
	}
	if cfg.EmbedModel != "text-embedding-3-small" {
		t.Errorf("Expected EmbedModel 'text-embedding-3-small', got %q", cfg.EmbedModel)
	}
	if cfg.Dim != 1536 {
		t.Errorf("Expected Dim 1536, got %d", cfg.Dim)
	}
	if cfg.Auth.Enabled != true {
		t.Errorf("Expected Auth.Enabled true, got %v", cfg.Auth.Enabled)
	}
	if cfg.Auth.GithubClientID != "test-client-id" {
		t.Errorf("Expected Auth.GithubClientID 'test-client-id', got %q", cfg.Auth.GithubClientID)
	}
}

func TestLoadFromEnvironmentVariables(t *testing.T) {
	clearTestEnv(t)

	// Set environment variables
	envVars := map[string]string{
		"REPOSEARCH_PROVIDER":                  "vertexai",
		"REPOSEARCH_PROVIDER_API_KEY":          "env-api-key",
		"REPOSEARCH_PROVIDER_EMBEDDING_MODEL":  "env-embed-model",
		"REPOSEARCH_PROVIDER_SUMMARY_MODEL":    "env-summary-model",
		"REPOSEARCH_PROVIDER_PROJECT_ID":       "env-project-id",
		"REPOSEARCH_PROVIDER_LOCATION":         "europe-west1",
		"REPOSEARCH_EMBED_DIM":                 "768",
		"REPOSEARCH_DB_URL":                    "postgres://env:env@localhost:5432/envdb",
		"REPOSEARCH_REPO_ROOT":                 "/env/repo",
		"REPOSEARCH_GIT_REPO":                  "https://github.com/env/repo.git",
		"REPOSEARCH_GITHUB_TOKEN":              "ghp_env123",
		"REPOSEARCH_GIT_REF":                   "feature",
		"REPOSEARCH_LOG_LEVEL":                 "warn",
		"REPOSEARCH_AUTH_ENABLED":              "true",
		"REPOSEARCH_AUTH_JWT_SECRET":           "env-jwt-secret",
		"REPOSEARCH_AUTH_GITHUB_CLIENT_ID":     "env-client-id",
		"REPOSEARCH_AUTH_GITHUB_CLIENT_SECRET": "env-client-secret",
		"REPOSEARCH_AUTH_GITHUB_REDIRECT_URL":  "https://env.com/auth/callback",
		"REPOSEARCH_AUTH_GITHUB_ALLOWED_ORG":   "env-org",
	}

	for key, value := range envVars {
		t.Setenv(key, value)
	}

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	cfg, err := Load("", fs)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify environment values were loaded
	if cfg.Provider != "vertexai" {
		t.Errorf("Expected Provider 'vertexai', got %q", cfg.Provider)
	}
	if cfg.APIKey != "env-api-key" {
		t.Errorf("Expected APIKey 'env-api-key', got %q", cfg.APIKey)
	}
	if cfg.Dim != 768 {
		t.Errorf("Expected Dim 768, got %d", cfg.Dim)
	}
	if cfg.Auth.Enabled != true {
		t.Errorf("Expected Auth.Enabled true, got %v", cfg.Auth.Enabled)
	}
	if cfg.Auth.GithubClientID != "env-client-id" {
		t.Errorf("Expected Auth.GithubClientID 'env-client-id', got %q", cfg.Auth.GithubClientID)
	}
}

func TestLoadFromFlags(t *testing.T) {
	clearTestEnv(t)

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	// Simulate command line arguments
	args := []string{
		"--provider", "google",
		"--provider-api-key", "flag-api-key",
		"--provider-embedding-model", "flag-embed-model",
		"--embed-dim", "2048",
		"--db-url", "postgres://flag:flag@localhost:5432/flagdb",
		"--auth-enabled",
		"--auth-github-client-id", "flag-client-id",
		"--log-level", "error",
	}

	// Save original os.Args and restore after test
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = append([]string{"test"}, args...)

	cfg, err := Load("", fs)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify flag values were loaded
	if cfg.Provider != "google" {
		t.Errorf("Expected Provider 'google', got %q", cfg.Provider)
	}
	if cfg.APIKey != "flag-api-key" {
		t.Errorf("Expected APIKey 'flag-api-key', got %q", cfg.APIKey)
	}
	if cfg.Dim != 2048 {
		t.Errorf("Expected Dim 2048, got %d", cfg.Dim)
	}
	if cfg.Auth.Enabled != true {
		t.Errorf("Expected Auth.Enabled true, got %v", cfg.Auth.Enabled)
	}
	if cfg.LogLevel != "error" {
		t.Errorf("Expected LogLevel 'error', got %q", cfg.LogLevel)
	}
}

func TestConfigPrecedence(t *testing.T) {
	// Test that flags override environment variables
	clearTestEnv(t)

	// Set environment variable
	t.Setenv("REPOSEARCH_PROVIDER", "env-provider")
	t.Setenv("REPOSEARCH_LOG_LEVEL", "env-level")

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	// Set flag to override environment
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"test", "--provider", "flag-provider"}

	cfg, err := Load("", fs)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Flag should override environment
	if cfg.Provider != "flag-provider" {
		t.Errorf("Expected Provider 'flag-provider' (flag should override env), got %q", cfg.Provider)
	}
	// Environment should be used where no flag is set
	if cfg.LogLevel != "env-level" {
		t.Errorf("Expected LogLevel 'env-level' (from env), got %q", cfg.LogLevel)
	}
}

func TestAutoDiscoverConfigFile(t *testing.T) {
	// Test auto-discovery of config files
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create a config file in auto-discovery location
	configContent := `provider: "discovered"`
	err := os.WriteFile("config.yaml", []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	clearTestEnv(t)
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	cfg, err := Load("", fs) // Empty path should trigger auto-discovery
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Provider != "discovered" {
		t.Errorf("Expected Provider 'discovered' (from auto-discovered file), got %q", cfg.Provider)
	}
}

func TestConfigFileFromEnvironment(t *testing.T) {
	// Test using REPOSEARCH_CONFIG environment variable
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "custom-config.yaml")

	configContent := `provider: "env-config"`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	clearTestEnv(t)
	t.Setenv("REPOSEARCH_CONFIG", configFile)

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	cfg, err := Load("", fs)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Provider != "env-config" {
		t.Errorf("Expected Provider 'env-config' (from REPOSEARCH_CONFIG), got %q", cfg.Provider)
	}
}

func TestValidation(t *testing.T) {
	// Test validation rules
	clearTestEnv(t)

	// Set an empty database URL to trigger validation error
	t.Setenv("REPOSEARCH_DB_URL", "   ") // Only whitespace

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	_, err := Load("", fs)
	if err == nil {
		t.Fatal("Expected validation error for empty database URL")
	}
	if !strings.Contains(err.Error(), "REPOSEARCH_DB_URL is required") {
		t.Errorf("Expected database URL validation error, got: %v", err)
	}
}

func TestInvalidYAMLFile(t *testing.T) {
	// Test error handling for invalid YAML
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `
provider: "test"
invalid: yaml: content: [
`

	err := os.WriteFile(configFile, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid YAML file: %v", err)
	}

	clearTestEnv(t)
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	_, err = Load(configFile, fs)
	if err == nil {
		t.Fatal("Expected error for invalid YAML file")
	}
	if !strings.Contains(err.Error(), "load yaml") {
		t.Errorf("Expected YAML load error, got: %v", err)
	}
}

func TestNonExistentConfigFile(t *testing.T) {
	// Test error handling for non-existent config file
	clearTestEnv(t)
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	_, err := Load("/non/existent/config.yaml", fs)
	if err == nil {
		t.Fatal("Expected error for non-existent config file")
	}
	if !strings.Contains(err.Error(), "config file not found") {
		t.Errorf("Expected: config file not found, got: %v", err)
	}
}

func TestFileExists(t *testing.T) {
	// Test fileExists helper function
	tmpDir := t.TempDir()

	// Test with existing file
	existingFile := filepath.Join(tmpDir, "existing.txt")
	err := os.WriteFile(existingFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !fileExists(existingFile) {
		t.Error("fileExists should return true for existing file")
	}

	// Test with non-existent file
	if fileExists(filepath.Join(tmpDir, "nonexistent.txt")) {
		t.Error("fileExists should return false for non-existent file")
	}

	// Test with directory
	if fileExists(tmpDir) {
		t.Error("fileExists should return false for directory")
	}
}

func TestLoadYAML(t *testing.T) {
	// Test loadYAML helper function
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "test.yaml")

	type TestStruct struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}

	yamlContent := `
name: "test"
value: 42
`

	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	var result TestStruct
	err = loadYAML(yamlFile, &result)
	if err != nil {
		t.Fatalf("loadYAML failed: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("Expected Name 'test', got %q", result.Name)
	}
	if result.Value != 42 {
		t.Errorf("Expected Value 42, got %d", result.Value)
	}

	// Test with non-existent file
	err = loadYAML("/non/existent/file.yaml", &result)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestBindFlagsAndApplyChangedFlags(t *testing.T) {
	// Test that bindFlags properly sets up all flags
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cfg := Specification{
		Provider: "initial",
		Dim:      1024,
		Auth: AuthSpecification{
			Enabled: false,
		},
	}

	bindFlags(fs, &cfg)

	// Verify that flags exist and have correct defaults
	providerFlag := fs.Lookup("provider")
	if providerFlag == nil {
		t.Fatal("provider flag not found")
	}
	if providerFlag.DefValue != "initial" {
		t.Errorf("Expected provider default 'initial', got %q", providerFlag.DefValue)
	}

	dimFlag := fs.Lookup("embed-dim")
	if dimFlag == nil {
		t.Fatal("embed-dim flag not found")
	}

	authEnabledFlag := fs.Lookup("auth-enabled")
	if authEnabledFlag == nil {
		t.Fatal("auth-enabled flag not found")
	}

	// Test applyChangedFlags
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"test", "--provider", "changed", "--embed-dim", "2048", "--auth-enabled"}

	err := fs.Parse(os.Args[1:])
	if err != nil {
		t.Fatalf("Flag parsing failed: %v", err)
	}

	applyChangedFlags(fs, &cfg)

	if cfg.Provider != "changed" {
		t.Errorf("Expected Provider 'changed', got %q", cfg.Provider)
	}
	if cfg.Dim != 2048 {
		t.Errorf("Expected Dim 2048, got %d", cfg.Dim)
	}
	if cfg.Auth.Enabled != true {
		t.Errorf("Expected Auth.Enabled true, got %v", cfg.Auth.Enabled)
	}
}

func TestLogLevelDefaulting(t *testing.T) {
	// Test that empty log level gets defaulted to "info"
	clearTestEnv(t)
	t.Setenv("REPOSEARCH_LOG_LEVEL", "")

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	cfg, err := Load("", fs)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected LogLevel to default to 'info' when empty, got %q", cfg.LogLevel)
	}
}

func TestInvalidFlagParsing(t *testing.T) {
	// Test error handling for invalid flag parsing
	clearTestEnv(t)

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	// Simulate invalid flags
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"test", "--embed-dim", "invalid-number"}

	_, err := Load("", fs)
	if err == nil {
		t.Fatal("Expected error for invalid flag value")
	}
	// The error should be related to flag parsing
	if !strings.Contains(err.Error(), "invalid argument") && !strings.Contains(err.Error(), "strconv.Atoi") {
		t.Logf("Got error (which is expected): %v", err)
	}
}

func TestEnvconfigProcessError(t *testing.T) {
	// This test is tricky because envconfig.Process rarely fails with valid structs
	// But we can test the error handling path by ensuring our test setup is correct
	clearTestEnv(t)

	// Set a malformed integer environment variable
	t.Setenv("REPOSEARCH_EMBED_DIM", "not-a-number")

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	_, err := Load("", fs)
	if err == nil {
		t.Fatal("Expected error for invalid integer in environment variable")
	}

	// Should contain error about envconfig or parsing
	if !strings.Contains(strings.ToLower(err.Error()), "env") && !strings.Contains(err.Error(), "parse") {
		t.Logf("Got error (which is expected): %v", err)
	}
}

func TestAllAutoDiscoveryPaths(t *testing.T) {
	// Test all auto-discovery paths one by one
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create config directory
	err := os.Mkdir("config", 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Test each auto-discovery path
	testCases := []struct {
		path     string
		content  string
		expected string
	}{
		{"config/reposearch.yaml", `provider: "reposearch-yaml"`, "reposearch-yaml"},
		{"config/config.yaml", `provider: "config-yaml"`, "config-yaml"},
		{"./reposearch.yaml", `provider: "dot-reposearch"`, "dot-reposearch"},
		{"./config.yaml", `provider: "dot-config"`, "dot-config"},
	}

	for i, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			// Clean up any existing files
			for _, otherCase := range testCases {
				if err := os.Remove(otherCase.path); err != nil && !os.IsNotExist(err) {
					t.Logf("Failed to remove %s: %v", otherCase.path, err)
				}
			}

			// Create the current test file
			err := os.WriteFile(tc.path, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			clearTestEnv(t)
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

			cfg, err := Load("", fs)
			if err != nil {
				t.Fatalf("Load failed for %s: %v", tc.path, err)
			}

			if cfg.Provider != tc.expected {
				t.Errorf("Test %d (%s): Expected Provider %q, got %q", i, tc.path, tc.expected, cfg.Provider)
			}
		})
	}
}

func TestAllFlagsAreBound(t *testing.T) {
	// Ensure all struct fields have corresponding flags
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cfg := Specification{}

	bindFlags(fs, &cfg)

	expectedFlags := []string{
		"config", "provider", "provider-api-key", "provider-embedding-model",
		"provider-summary-model", "provider-project-id", "provider-location",
		"embed-dim", "db-url", "repo-root", "git-repo", "github-token",
		"git-ref", "log-level", "auth-enabled", "auth-jwt-secret",
		"auth-github-client-id", "auth-github-client-secret",
		"auth-github-redirect-url", "auth-github-allowed-org",
	}

	for _, flagName := range expectedFlags {
		if fs.Lookup(flagName) == nil {
			t.Errorf("Flag %q not found", flagName)
		}
	}
}

// Helper function to clear test environment variables
func clearTestEnv(t *testing.T) {
	t.Helper()

	envVars := []string{
		"REPOSEARCH_CONFIG",
		"REPOSEARCH_PROVIDER",
		"REPOSEARCH_PROVIDER_API_KEY",
		"REPOSEARCH_PROVIDER_EMBEDDING_MODEL",
		"REPOSEARCH_PROVIDER_SUMMARY_MODEL",
		"REPOSEARCH_PROVIDER_PROJECT_ID",
		"REPOSEARCH_PROVIDER_LOCATION",
		"REPOSEARCH_EMBED_DIM",
		"REPOSEARCH_DB_URL",
		"REPOSEARCH_REPO_ROOT",
		"REPOSEARCH_GIT_REPO",
		"REPOSEARCH_GITHUB_TOKEN",
		"REPOSEARCH_GIT_REF",
		"REPOSEARCH_LOG_LEVEL",
		"REPOSEARCH_AUTH_ENABLED",
		"REPOSEARCH_AUTH_JWT_SECRET",
		"REPOSEARCH_AUTH_GITHUB_CLIENT_ID",
		"REPOSEARCH_AUTH_GITHUB_CLIENT_SECRET",
		"REPOSEARCH_AUTH_GITHUB_REDIRECT_URL",
		"REPOSEARCH_AUTH_GITHUB_ALLOWED_ORG",
	}

	for _, envVar := range envVars {
		if err := os.Unsetenv(envVar); err != nil {
			t.Logf("Failed to unset environment variable %s: %v", envVar, err)
		}
	}
}

// Benchmark tests
func BenchmarkLoad(b *testing.B) {
	clearTestEnvBench(b)

	for i := 0; i < b.N; i++ {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		_, err := Load("", fs)
		if err != nil {
			b.Fatalf("Load failed: %v", err)
		}
	}
}

func BenchmarkLoadWithYAML(b *testing.B) {
	tmpDir := b.TempDir()
	configFile := filepath.Join(tmpDir, "bench-config.yaml")

	yamlContent := `
provider: "openai"
providerApiKey: "test-key"
embed-dim: 1536
`

	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write config file: %v", err)
	}

	clearTestEnvBench(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		_, err := Load(configFile, fs)
		if err != nil {
			b.Fatalf("Load failed: %v", err)
		}
	}
}

func clearTestEnvBench(b *testing.B) {
	b.Helper()

	envVars := []string{
		"REPOSEARCH_CONFIG", "REPOSEARCH_PROVIDER", "REPOSEARCH_PROVIDER_API_KEY",
		"REPOSEARCH_PROVIDER_EMBEDDING_MODEL", "REPOSEARCH_PROVIDER_SUMMARY_MODEL",
		"REPOSEARCH_PROVIDER_PROJECT_ID", "REPOSEARCH_PROVIDER_LOCATION",
		"REPOSEARCH_EMBED_DIM", "REPOSEARCH_DB_URL", "REPOSEARCH_REPO_ROOT",
		"REPOSEARCH_GIT_REPO", "REPOSEARCH_GITHUB_TOKEN", "REPOSEARCH_GIT_REF",
		"REPOSEARCH_LOG_LEVEL", "REPOSEARCH_AUTH_ENABLED", "REPOSEARCH_AUTH_JWT_SECRET",
		"REPOSEARCH_AUTH_GITHUB_CLIENT_ID", "REPOSEARCH_AUTH_GITHUB_CLIENT_SECRET",
		"REPOSEARCH_AUTH_GITHUB_REDIRECT_URL", "REPOSEARCH_AUTH_GITHUB_ALLOWED_ORG",
	}

	for _, envVar := range envVars {
		if err := os.Unsetenv(envVar); err != nil {
			// Ignore errors in benchmark cleanup
			_ = err
		}
	}
}
