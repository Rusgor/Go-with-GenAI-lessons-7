package handler

import (
	"net/http"
	"testing"

	"homework/internal/model"
)

func TestRBACBookCreationByRole(t *testing.T) {
	t.Setenv("JWT_SECRET", securityTestSecret)
	router := securityRouter(newMemoryUsers())
	payload := `{"title":"Safe Book","isbn":"9780000000999","category_id":1,"author_ids":[1]}`
	tests := []struct {
		role       string
		wantStatus int
	}{
		{role: model.UserReader, wantStatus: http.StatusForbidden},
		{role: model.UserLibrarian, wantStatus: http.StatusCreated},
		{role: model.UserAdmin, wantStatus: http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			response := securityRequest(router, http.MethodPost, "/api/v1/books", payload, securityToken(t, 10, tt.role))
			if response.Code != tt.wantStatus {
				t.Fatalf("book creation for %s = %d, want %d: %s", tt.role, response.Code, tt.wantStatus, response.Body.String())
			}
		})
	}
}

func TestRBACRejectsMissingAndExpiredTokens(t *testing.T) {
	t.Setenv("JWT_SECRET", securityTestSecret)
	router := securityRouter(newMemoryUsers())
	payload := `{"title":"Safe Book","isbn":"9780000000999","category_id":1,"author_ids":[1]}`
	tests := []struct {
		name  string
		token string
	}{
		{name: "missing token"},
		{name: "expired token", token: expiredSecurityToken(t)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := securityRequest(router, http.MethodPost, "/api/v1/books", payload, tt.token)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("protected request = %d, want 401: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRBACAdminOnlyRouteByRole(t *testing.T) {
	t.Setenv("JWT_SECRET", securityTestSecret)
	router := securityRouter(newMemoryUsers())
	tests := []struct {
		role       string
		wantStatus int
	}{
		{role: model.UserReader, wantStatus: http.StatusForbidden},
		{role: model.UserLibrarian, wantStatus: http.StatusForbidden},
		{role: model.UserAdmin, wantStatus: http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			response := securityRequest(router, http.MethodGet, "/api/v1/admin-check", "", securityToken(t, 42, tt.role))
			if response.Code != tt.wantStatus {
				t.Fatalf("admin route for %s = %d, want %d: %s", tt.role, response.Code, tt.wantStatus, response.Body.String())
			}
		})
	}
}
