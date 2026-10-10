package app_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"homework/internal/app"
)

func TestLibraryRouterHealthSwaggerAndCORS(t *testing.T) {
	router := app.NewLibraryRouter(nil)
	for _, path := range []string{"/health", "/swagger/index.html"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", path, response.Code)
		}
		if response.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Errorf("GET %s did not include CORS origin", path)
		}
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/books", nil)
	request.Header.Set("Origin", "http://example.com")
	request.Header.Set("Access-Control-Request-Method", "POST")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || !strings.Contains(response.Header().Get("Access-Control-Allow-Methods"), "POST") {
		t.Fatalf("OPTIONS preflight = %d, methods %q; want 204 with POST", response.Code, response.Header().Get("Access-Control-Allow-Methods"))
	}
}
