package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"homework/internal/middleware"
	"homework/internal/model"
)

func TestAuthRegistrationDuplicateAndRoleAssignment(t *testing.T) {
	t.Setenv("JWT_SECRET", securityTestSecret)
	users := newMemoryUsers()
	router := securityRouter(users)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "registration ignores caller role and hides password hash",
			body:       `{"email":"Reader@example.com","password":"correct horse battery staple","role":"admin"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "duplicate email is rejected",
			body:       `{"email":"reader@example.com","password":"correct horse battery staple"}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := securityRequest(router, http.MethodPost, "/api/v1/auth/register", tt.body, "")
			if response.Code != tt.wantStatus {
				t.Fatalf("register = %d, want %d: %s", response.Code, tt.wantStatus, response.Body.String())
			}
			if tt.wantStatus == http.StatusCreated && (strings.Contains(response.Body.String(), "password_hash") || !strings.Contains(response.Body.String(), `"role":"reader"`)) {
				t.Fatalf("registration leaked hash or accepted caller role: %s", response.Body.String())
			}
		})
	}
}

func TestAuthSuccessfulLoginIssuesExpectedJWT(t *testing.T) {
	t.Setenv("JWT_SECRET", securityTestSecret)
	router := securityRouter(newMemoryUsers())
	registered := securityRequest(router, http.MethodPost, "/api/v1/auth/register", `{"email":"reader@example.com","password":"correct horse battery staple"}`, "")
	if registered.Code != http.StatusCreated {
		t.Fatalf("register = %d, want 201: %s", registered.Code, registered.Body.String())
	}

	issuedAt := time.Now()
	response := securityRequest(router, http.MethodPost, "/api/v1/auth/login", `{"email":"reader@example.com","password":"correct horse battery staple"}`, "")
	if response.Code != http.StatusOK {
		t.Fatalf("login = %d, want 200: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "password_hash") {
		t.Fatalf("login response exposed password hash: %s", response.Body.String())
	}
	var loginBody struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &loginBody); err != nil || loginBody.AccessToken == "" {
		t.Fatalf("login did not return token: %s", response.Body.String())
	}

	claims := &middleware.TokenClaims{}
	parsed, err := jwt.ParseWithClaims(loginBody.AccessToken, claims, func(token *jwt.Token) (any, error) {
		return []byte(securityTestSecret), nil
	})
	if err != nil || parsed == nil || !parsed.Valid || parsed.Method.Alg() != jwt.SigningMethodHS256.Alg() {
		t.Fatalf("issued token invalid or not HS256: %v", err)
	}
	if claims.UserID != 1 || claims.Role != model.UserReader {
		t.Fatalf("claims user_id/role = %d/%q, want 1/%q", claims.UserID, claims.Role, model.UserReader)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("JWT is missing exp claim")
	}
	wantExpiry := issuedAt.Add(24 * time.Hour)
	if delta := claims.ExpiresAt.Time.Sub(wantExpiry); delta < -time.Second || delta > time.Second {
		t.Fatalf("JWT exp = %s, want approximately %s", claims.ExpiresAt.Time, wantExpiry)
	}
}

func TestAuthBadCredentialsUseSameGenericResponse(t *testing.T) {
	t.Setenv("JWT_SECRET", securityTestSecret)
	router := securityRouter(newMemoryUsers())
	registered := securityRequest(router, http.MethodPost, "/api/v1/auth/register", `{"email":"reader@example.com","password":"correct horse battery staple"}`, "")
	if registered.Code != http.StatusCreated {
		t.Fatalf("register = %d, want 201: %s", registered.Code, registered.Body.String())
	}

	tests := []struct {
		name  string
		body  string
		other string
	}{
		{
			name:  "wrong password",
			body:  `{"email":"reader@example.com","password":"not-the-password"}`,
			other: `{"email":"missing@example.com","password":"not-the-password"}`,
		},
		{
			name:  "unknown email",
			body:  `{"email":"missing@example.com","password":"not-the-password"}`,
			other: `{"email":"reader@example.com","password":"not-the-password"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := securityRequest(router, http.MethodPost, "/api/v1/auth/login", tt.body, "")
			comparison := securityRequest(router, http.MethodPost, "/api/v1/auth/login", tt.other, "")
			if response.Code != http.StatusUnauthorized || response.Body.String() != comparison.Body.String() || !strings.Contains(response.Body.String(), "invalid credentials") {
				t.Fatalf("credentials response is not the same generic 401: got %d %s; comparison %d %s", response.Code, response.Body.String(), comparison.Code, comparison.Body.String())
			}
		})
	}
}
