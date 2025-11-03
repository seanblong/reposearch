package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestInitializeAuth(t *testing.T) {
	// Test initialization
	InitializeAuth("test-secret", "client-id", "client-secret", "http://localhost/callback", "test-org", true)

	if authConfig == nil {
		t.Fatal("authConfig should not be nil after initialization")
	}

	if string(authConfig.JwtSecret) != "test-secret" {
		t.Errorf("Expected JwtSecret 'test-secret', got %q", string(authConfig.JwtSecret))
	}
	if authConfig.ClientID != "client-id" {
		t.Errorf("Expected ClientID 'client-id', got %q", authConfig.ClientID)
	}
	if authConfig.ClientSecret != "client-secret" {
		t.Errorf("Expected ClientSecret 'client-secret', got %q", authConfig.ClientSecret)
	}
	if authConfig.RedirectURL != "http://localhost/callback" {
		t.Errorf("Expected RedirectURL 'http://localhost/callback', got %q", authConfig.RedirectURL)
	}
	if authConfig.AllowedOrg != "test-org" {
		t.Errorf("Expected AllowedOrg 'test-org', got %q", authConfig.AllowedOrg)
	}
	if !authConfig.Enabled {
		t.Error("Expected Enabled to be true")
	}
}

func TestIsAuthEnabled(t *testing.T) {
	// Test when auth config is nil
	authConfig = nil
	if IsAuthEnabled() {
		t.Error("Expected IsAuthEnabled to return false when authConfig is nil")
	}

	// Test when auth is disabled
	InitializeAuth("secret", "id", "secret", "url", "", false)
	if IsAuthEnabled() {
		t.Error("Expected IsAuthEnabled to return false when auth is disabled")
	}

	// Test when auth is enabled
	InitializeAuth("secret", "id", "secret", "url", "", true)
	if !IsAuthEnabled() {
		t.Error("Expected IsAuthEnabled to return true when auth is enabled")
	}
}

func TestGenerateState(t *testing.T) {
	state1 := GenerateState()
	state2 := GenerateState()

	// States should be different
	if state1 == state2 {
		t.Error("GenerateState should produce different values")
	}

	// States should be base64 encoded (roughly 32 bytes -> 44 chars when base64 encoded)
	if len(state1) == 0 {
		t.Error("GenerateState should not return empty string")
	}

	// Should be valid base64
	if strings.Contains(state1, " ") {
		t.Error("State should not contain spaces")
	}
}

func TestGetGithubLoginURL(t *testing.T) {
	// Test when authConfig is nil
	authConfig = nil
	url := GetGithubLoginURL("test-state")
	if url != "" {
		t.Error("Expected empty URL when authConfig is nil")
	}

	// Test with basic config (no org)
	InitializeAuth("secret", "test-client-id", "client-secret", "http://localhost/callback", "", true)
	url = GetGithubLoginURL("test-state")

	expected := "https://github.com/login/oauth/authorize?client_id=test-client-id&redirect_uri=http://localhost/callback&scope=read:user,user:email&state=test-state"
	if url != expected {
		t.Errorf("Expected URL %q, got %q", expected, url)
	}

	// Test with org restriction
	InitializeAuth("secret", "test-client-id", "client-secret", "http://localhost/callback", "test-org", true)
	url = GetGithubLoginURL("test-state")

	expected = "https://github.com/login/oauth/authorize?client_id=test-client-id&redirect_uri=http://localhost/callback&scope=read:user,user:email,read:org&state=test-state"
	if url != expected {
		t.Errorf("Expected URL with org scope %q, got %q", expected, url)
	}
}

func TestExchangeCodeForToken(t *testing.T) {
	// Test when authConfig is nil
	authConfig = nil
	_, err := ExchangeCodeForToken("test-code")
	if err == nil {
		t.Error("Expected error when authConfig is nil")
	}
	if !strings.Contains(err.Error(), "auth not initialized") {
		t.Errorf("Expected 'auth not initialized' error, got: %v", err)
	}

	// Mock Github's token exchange endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept header 'application/json', got %q", r.Header.Get("Accept"))
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type 'application/x-www-form-urlencoded', got %q", r.Header.Get("Content-Type"))
		}

		// Return successful response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-access-token",
			"token_type":   "bearer",
		}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// Test successful token exchange (we'd need to mock the HTTP client or patch the URL)
	// For now, let's test the error case with a real request that will fail
	InitializeAuth("secret", "test-client", "test-secret", "http://localhost/callback", "", true)

	// This will make a real HTTP request and likely fail, which is expected for testing
	token, err := ExchangeCodeForToken("invalid-code")
	if err == nil {
		t.Error("Expected error for invalid code")
	}
	if token != "" {
		t.Error("Expected empty token on error")
	}
}

func TestGetGithubUser(t *testing.T) {
	// Mock Github API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Errorf("Expected Bearer token in Authorization header")
		}
		if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
			t.Errorf("Expected Github API Accept header")
		}

		// Return mock user data
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(GithubUser{
			Login:     "testuser",
			Name:      "Test User",
			Email:     "test@example.com",
			AvatarURL: "https://github.com/avatar.jpg",
		}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// Since we can't easily mock the HTTP client, let's test with invalid token
	// This will make a real request and fail
	InitializeAuth("secret", "client", "secret", "url", "", true)

	user, err := GetGithubUser("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
	if user != nil {
		t.Error("Expected nil user on error")
	}
}

func TestIsOrgMember(t *testing.T) {
	// Mock Github org membership API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if URL matches org membership endpoint
		if !strings.Contains(r.URL.Path, "/orgs/") || !strings.Contains(r.URL.Path, "/members/") {
			t.Error("Expected org membership API endpoint")
		}

		// Return 200 for member, 404 for non-member
		if strings.Contains(r.URL.Path, "member-user") {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	// This will test with real Github API and likely fail
	// In a real test, we'd mock the HTTP client
	isMember := isOrgMember("invalid-token", "testuser", "testorg")
	if isMember {
		t.Error("Expected false for invalid token/org")
	}
}

func TestGenerateJWT(t *testing.T) {
	// Test when authConfig is nil
	authConfig = nil
	user := &GithubUser{Login: "testuser", Name: "Test User"}
	_, err := GenerateJWT(user)
	if err == nil {
		t.Error("Expected error when authConfig is nil")
	}

	// Test successful JWT generation
	InitializeAuth("test-secret-key", "client", "secret", "url", "", true)

	user = &GithubUser{
		Login:     "testuser",
		Name:      "Test User",
		Email:     "test@example.com",
		AvatarURL: "https://avatar.jpg",
	}

	tokenString, err := GenerateJWT(user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	if tokenString == "" {
		t.Error("Expected non-empty JWT token")
	}

	// Verify the token can be parsed
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return authConfig.JwtSecret, nil
	})

	if err != nil {
		t.Fatalf("Failed to parse generated JWT: %v", err)
	}

	if !token.Valid {
		t.Error("Generated JWT should be valid")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		t.Fatal("Failed to parse claims")
	}

	if claims.Login != user.Login {
		t.Errorf("Expected login %q, got %q", user.Login, claims.Login)
	}
	if claims.Name != user.Name {
		t.Errorf("Expected name %q, got %q", user.Name, claims.Name)
	}
	if claims.Email != user.Email {
		t.Errorf("Expected email %q, got %q", user.Email, claims.Email)
	}
	if claims.AvatarURL != user.AvatarURL {
		t.Errorf("Expected avatar URL %q, got %q", user.AvatarURL, claims.AvatarURL)
	}
	if claims.Subject != user.Login {
		t.Errorf("Expected subject %q, got %q", user.Login, claims.Subject)
	}
}

func TestValidateJWT(t *testing.T) {
	// Test when authConfig is nil
	authConfig = nil
	_, err := ValidateJWT("some-token")
	if err == nil {
		t.Error("Expected error when authConfig is nil")
	}

	InitializeAuth("test-secret-key", "client", "secret", "url", "", true)

	// Test with invalid token
	_, err = ValidateJWT("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}

	// Test with valid token
	user := &GithubUser{
		Login:     "testuser",
		Name:      "Test User",
		Email:     "test@example.com",
		AvatarURL: "https://avatar.jpg",
	}

	tokenString, err := GenerateJWT(user)
	if err != nil {
		t.Fatalf("Failed to generate JWT for testing: %v", err)
	}

	validatedUser, err := ValidateJWT(tokenString)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	if validatedUser.Login != user.Login {
		t.Errorf("Expected login %q, got %q", user.Login, validatedUser.Login)
	}
	if validatedUser.Name != user.Name {
		t.Errorf("Expected name %q, got %q", user.Name, validatedUser.Name)
	}
	if validatedUser.Email != user.Email {
		t.Errorf("Expected email %q, got %q", user.Email, validatedUser.Email)
	}
	if validatedUser.AvatarURL != user.AvatarURL {
		t.Errorf("Expected avatar URL %q, got %q", user.AvatarURL, validatedUser.AvatarURL)
	}

	// Test with expired token
	expiredClaims := Claims{
		Login:     "testuser",
		Name:      "Test User",
		Email:     "test@example.com",
		AvatarURL: "https://avatar.jpg",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Subject:   "testuser",
		},
	}

	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, err := expiredToken.SignedString(authConfig.JwtSecret)
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	_, err = ValidateJWT(expiredTokenString)
	if err == nil {
		t.Error("Expected error for expired token")
	}

	// Test with wrong signing key
	wrongKey := []byte("wrong-key")
	wrongToken := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{Login: "testuser"})
	wrongTokenString, _ := wrongToken.SignedString(wrongKey)

	_, err = ValidateJWT(wrongTokenString)
	if err == nil {
		t.Error("Expected error for token with wrong signing key")
	}
}

func TestOptionalAuthMiddleware(t *testing.T) {
	// Test handler that records if it was called
	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(200)
		if _, err := w.Write([]byte("OK")); err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
	})

	// Test with auth disabled
	InitializeAuth("secret", "client", "secret", "url", "", false)
	middleware := OptionalAuthMiddleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handlerCalled = false
	middleware.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Handler should be called when auth is disabled")
	}
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test with auth enabled but no token
	InitializeAuth("secret", "client", "secret", "url", "", true)
	middleware = OptionalAuthMiddleware(testHandler)

	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()

	handlerCalled = false
	middleware.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("Handler should not be called when auth is enabled and no token provided")
	}
	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Authentication required") {
		t.Error("Expected authentication required message")
	}

	// Test with valid token in Authorization header
	user := &GithubUser{Login: "testuser", Name: "Test User"}
	tokenString, err := GenerateJWT(user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w = httptest.NewRecorder()

	handlerCalled = false
	middleware.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Handler should be called with valid token")
	}
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test with valid token in cookie
	req = httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: tokenString})
	w = httptest.NewRecorder()

	handlerCalled = false
	middleware.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Handler should be called with valid token in cookie")
	}
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test with invalid token
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w = httptest.NewRecorder()

	handlerCalled = false
	middleware.ServeHTTP(w, req)

	if handlerCalled {
		t.Error("Handler should not be called with invalid token")
	}
	if w.Code != 401 {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Invalid authentication token") {
		t.Error("Expected invalid token message")
	}
}

func TestGetUserFromContext(t *testing.T) {
	// Test with no user in context
	req := httptest.NewRequest("GET", "/test", nil)
	user := GetUserFromContext(req)
	if user != nil {
		t.Error("Expected nil user when not in context")
	}

	// Test with user in context
	testUser := &GithubUser{Login: "testuser", Name: "Test User"}
	ctx := context.WithValue(req.Context(), UserContextKey, testUser)
	req = req.WithContext(ctx)

	user = GetUserFromContext(req)
	if user == nil {
		t.Fatal("Expected user from context")
	}
	if user.Login != testUser.Login {
		t.Errorf("Expected user login %q, got %q", testUser.Login, user.Login)
	}

	// Test with wrong type in context
	ctx = context.WithValue(req.Context(), UserContextKey, "not-a-user")
	req = req.WithContext(ctx)

	user = GetUserFromContext(req)
	if user != nil {
		t.Error("Expected nil user when wrong type in context")
	}
}

func TestJWTTokenExpiration(t *testing.T) {
	InitializeAuth("test-secret", "client", "secret", "url", "", true)

	user := &GithubUser{Login: "testuser", Name: "Test User"}
	tokenString, err := GenerateJWT(user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	// Parse the token to check expiration
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return authConfig.JwtSecret, nil
	})
	if err != nil {
		t.Fatalf("Failed to parse JWT: %v", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		t.Fatal("Failed to parse claims")
	}

	// Check that expiration is set to 24 hours from now (with some tolerance)
	expectedExpiry := time.Now().Add(24 * time.Hour)
	actualExpiry := claims.ExpiresAt.Time

	diff := actualExpiry.Sub(expectedExpiry)
	if diff > time.Minute || diff < -time.Minute {
		t.Errorf("Token expiry should be ~24 hours from now, got %v", actualExpiry)
	}

	// Check that issued at is around now
	issuedAt := claims.IssuedAt.Time
	issuedDiff := time.Since(issuedAt)
	if issuedDiff > time.Minute || issuedDiff < 0 {
		t.Errorf("Token issued at should be around now, got %v", issuedAt)
	}
}

func TestAuthResponseSerialization(t *testing.T) {
	// Test AuthResponse JSON serialization
	user := GithubUser{
		Login:     "testuser",
		Name:      "Test User",
		Email:     "test@example.com",
		AvatarURL: "https://avatar.jpg",
	}

	response := AuthResponse{
		User:  user,
		Token: "test-token",
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal AuthResponse: %v", err)
	}

	var unmarshaled AuthResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal AuthResponse: %v", err)
	}

	if unmarshaled.User.Login != user.Login {
		t.Errorf("Expected login %q, got %q", user.Login, unmarshaled.User.Login)
	}
	if unmarshaled.Token != "test-token" {
		t.Errorf("Expected token 'test-token', got %q", unmarshaled.Token)
	}
}

func TestClaimsSerialization(t *testing.T) {
	// Test Claims JSON serialization
	claims := Claims{
		Login:     "testuser",
		Name:      "Test User",
		Email:     "test@example.com",
		AvatarURL: "https://avatar.jpg",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "testuser",
		},
	}

	data, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("Failed to marshal Claims: %v", err)
	}

	var unmarshaled Claims
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Claims: %v", err)
	}

	if unmarshaled.Login != claims.Login {
		t.Errorf("Expected login %q, got %q", claims.Login, unmarshaled.Login)
	}
	if unmarshaled.Subject != claims.Subject {
		t.Errorf("Expected subject %q, got %q", claims.Subject, unmarshaled.Subject)
	}
}

func TestGetEnvFunction(t *testing.T) {
	// Test getEnv helper function

	// Test with existing environment variable
	t.Setenv("TEST_AUTH_VAR", "test-value")
	value := getEnv("TEST_AUTH_VAR", "default")
	if value != "test-value" {
		t.Errorf("Expected 'test-value', got %q", value)
	}

	// Test with non-existing environment variable
	value = getEnv("NON_EXISTENT_VAR", "default-value")
	if value != "default-value" {
		t.Errorf("Expected 'default-value', got %q", value)
	}

	// Test with empty environment variable
	t.Setenv("EMPTY_VAR", "")
	value = getEnv("EMPTY_VAR", "default")
	if value != "default" {
		t.Errorf("Expected 'default' for empty env var, got %q", value)
	}
}

// Integration test that combines multiple auth functions
func TestAuthIntegration(t *testing.T) {
	// Initialize auth
	InitializeAuth("integration-secret", "client-id", "client-secret", "http://localhost/callback", "", true)

	// Create a user
	user := &GithubUser{
		Login:     "integrationuser",
		Name:      "Integration User",
		Email:     "integration@example.com",
		AvatarURL: "https://integration.jpg",
	}

	// Generate JWT
	tokenString, err := GenerateJWT(user)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	// Validate JWT
	validatedUser, err := ValidateJWT(tokenString)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	// Verify user data matches
	if validatedUser.Login != user.Login {
		t.Errorf("User data mismatch after JWT round-trip")
	}

	// Test middleware with this token
	handlerCalled := false
	var contextUser *GithubUser

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		contextUser = GetUserFromContext(r)
		w.WriteHeader(200)
	})

	middleware := OptionalAuthMiddleware(testHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Handler should be called with valid JWT")
	}
	if contextUser == nil {
		t.Fatal("User should be in context")
	}
	if contextUser.Login != user.Login {
		t.Errorf("Context user login mismatch: expected %q, got %q", user.Login, contextUser.Login)
	}
}

// Benchmark tests
func BenchmarkGenerateJWT(b *testing.B) {
	InitializeAuth("benchmark-secret", "client", "secret", "url", "", true)
	user := &GithubUser{Login: "benchuser", Name: "Bench User"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateJWT(user)
		if err != nil {
			b.Fatalf("Failed to generate JWT: %v", err)
		}
	}
}

func BenchmarkValidateJWT(b *testing.B) {
	InitializeAuth("benchmark-secret", "client", "secret", "url", "", true)
	user := &GithubUser{Login: "benchuser", Name: "Bench User"}

	tokenString, err := GenerateJWT(user)
	if err != nil {
		b.Fatalf("Failed to generate JWT for benchmark: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ValidateJWT(tokenString)
		if err != nil {
			b.Fatalf("Failed to validate JWT: %v", err)
		}
	}
}

func BenchmarkGenerateState(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateState()
	}
}
