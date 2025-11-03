package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const UserContextKey ContextKey = "user"

type GithubUser struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type AuthResponse struct {
	User  GithubUser `json:"user"`
	Token string     `json:"token,omitempty"`
}

type Claims struct {
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	jwt.RegisteredClaims
}

var (
	authConfig *AuthConfig
)

type AuthConfig struct {
	JwtSecret    []byte
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AllowedOrg   string
	Enabled      bool
}

// InitializeAuth sets up the auth configuration
func InitializeAuth(jwtSecret, clientID, clientSecret, redirectURL, allowedOrg string, enabled bool) {
	authConfig = &AuthConfig{
		JwtSecret:    []byte(jwtSecret),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AllowedOrg:   allowedOrg,
		Enabled:      enabled,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsAuthEnabled returns whether authentication is enabled
func IsAuthEnabled() bool {
	if authConfig == nil {
		return false
	}
	return authConfig.Enabled
}

// GenerateState creates a random state parameter for OAuth
func GenerateState() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fall back to a predictable state in case of error
		// This should rarely happen, but provides a safer fallback
		return "fallback-state-" + fmt.Sprintf("%d", time.Now().Unix())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// GetGithubLoginURL returns the Github OAuth login URL
func GetGithubLoginURL(state string) string {
	if authConfig == nil {
		return ""
	}
	scope := "read:user,user:email"
	if authConfig.AllowedOrg != "" {
		scope += ",read:org"
	}
	return fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		authConfig.ClientID, authConfig.RedirectURL, scope, state,
	)
}

// ExchangeCodeForToken exchanges OAuth code for access token
func ExchangeCodeForToken(code string) (string, error) {
	if authConfig == nil {
		return "", errors.New("auth not initialized")
	}
	data := fmt.Sprintf(
		"client_id=%s&client_secret=%s&code=%s",
		authConfig.ClientID, authConfig.ClientSecret, code,
	)

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Failed to close response body: %v\n", err)
		}
	}()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if accessToken, ok := result["access_token"].(string); ok {
		return accessToken, nil
	}

	return "", fmt.Errorf("failed to get access token")
}

// GetGithubUser fetches user info from Github API
func GetGithubUser(accessToken string) (*GithubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Failed to close response body: %v\n", err)
		}
	}()

	// Check for HTTP error status
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var user GithubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	// Check org membership if required
	if authConfig.AllowedOrg != "" {
		if !isOrgMember(accessToken, user.Login, authConfig.AllowedOrg) {
			return nil, fmt.Errorf("user is not a member of the required organization")
		}
	}

	return &user, nil
}

// isOrgMember checks if user is a member of the specified organization
func isOrgMember(accessToken, username, org string) bool {
	url := fmt.Sprintf("https://api.github.com/orgs/%s/members/%s", org, username)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Failed to close response body: %v\n", err)
		}
	}()

	// 204 means user is a public member, 200 means private member
	return resp.StatusCode == 200 || resp.StatusCode == 204
}

// GenerateJWT creates a JWT token for the user
func GenerateJWT(user *GithubUser) (string, error) {
	if authConfig == nil {
		return "", errors.New("auth not initialized")
	}
	claims := Claims{
		Login:     user.Login,
		Name:      user.Name,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.Login,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(authConfig.JwtSecret)
}

// ValidateJWT validates and parses a JWT token
func ValidateJWT(tokenString string) (*GithubUser, error) {
	if authConfig == nil {
		return nil, errors.New("auth not initialized")
	}
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return authConfig.JwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return &GithubUser{
			Login:     claims.Login,
			Name:      claims.Name,
			Email:     claims.Email,
			AvatarURL: claims.AvatarURL,
		}, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// OptionalAuthMiddleware extracts and validates JWT from request if auth is enabled
// If auth is disabled, it allows all requests through
func OptionalAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If auth is disabled, just pass through
		if !IsAuthEnabled() {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header or cookie
		var tokenString string

		// Try Authorization header first
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Try cookie
			if cookie, err := r.Cookie("auth_token"); err == nil {
				tokenString = cookie.Value
			}
		}

		if tokenString == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		user, err := ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Invalid authentication token", http.StatusUnauthorized)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetUserFromContext extracts user from request context
func GetUserFromContext(r *http.Request) *GithubUser {
	if user, ok := r.Context().Value(UserContextKey).(*GithubUser); ok {
		return user
	}
	return nil
}
