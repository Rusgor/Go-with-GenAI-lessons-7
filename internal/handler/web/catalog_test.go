package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	webhandler "homework/internal/handler/web"
	"homework/internal/middleware"
	"homework/internal/model"
	"homework/internal/repository"
	assets "homework/web"
)

const testSecret = "unit-test-secret-with-sufficient-entropy"

type fakeBooks struct {
	mu      sync.Mutex
	books   []model.Book
	deleted []uint64
}

func (f *fakeBooks) List(_ context.Context, filter repository.BookFilter) ([]model.Book, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []model.Book{}
	for _, b := range f.books {
		if filter.Query == "" || strings.Contains(strings.ToLower(b.Title), strings.ToLower(filter.Query)) {
			out = append(out, b)
		}
	}
	return out, nil
}

func (f *fakeBooks) Delete(_ context.Context, id uint64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, b := range f.books {
		if b.ID == id {
			f.deleted = append(f.deleted, id)
			return nil
		}
	}
	return repository.ErrNotFound
}

type fakeCopies struct{}

func (fakeCopies) AvailabilityByBooks(_ context.Context, ids []uint64) (map[uint64]repository.BookAvailability, error) {
	out := map[uint64]repository.BookAvailability{}
	for _, id := range ids {
		if id == 1 {
			out[id] = repository.BookAvailability{Total: 3, Available: 2}
		} else {
			out[id] = repository.BookAvailability{Total: 1, Available: 0}
		}
	}
	return out, nil
}

type fakeUsers struct{ user *model.User }

func (f fakeUsers) FindByEmail(_ context.Context, email string) (*model.User, error) {
	if f.user != nil && f.user.Email == email {
		return f.user, nil
	}
	return nil, repository.ErrNotFound
}

func newRouter(t *testing.T, books *fakeBooks, users fakeUsers) http.Handler {
	t.Helper()
	t.Setenv("JWT_SECRET", testSecret)
	gin.SetMode(gin.TestMode)
	pages, err := assets.Templates()
	if err != nil {
		t.Fatalf("embedded templates: %v", err)
	}
	router := gin.New()
	webhandler.New(books, fakeCopies{}, users, pages).Register(router)
	return router
}

func seedBooks() *fakeBooks {
	return &fakeBooks{books: []model.Book{
		{ID: 1, Title: "Go in Practice", ISBN: "111", Category: model.Category{Name: "Technology"}, Authors: []model.Author{{FullName: "Jon Bell"}}},
		{ID: 2, Title: "Python Basics", ISBN: "222", Category: model.Category{Name: "Technology"}, Authors: []model.Author{{FullName: "Ada North"}}},
		{ID: 3, Title: `<script>alert(1)</script>`, ISBN: "333", Category: model.Category{Name: "Fiction"}},
	}}
}

func tokenFor(t *testing.T, role string) string {
	t.Helper()
	token, err := middleware.GenerateToken(model.User{ID: 7, Role: role}, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func do(router http.Handler, method, target string, headers map[string]string, cookie string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: webhandler.CookieName, Value: cookie})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestCatalogPageAndFragments(t *testing.T) {
	htmx := map[string]string{"HX-Request": "true"}
	tests := []struct {
		name     string
		target   string
		headers  map[string]string
		role     string
		status   int
		contains []string
		excludes []string
	}{
		{
			name: "full catalog page has layout and book table", target: "/catalog", status: 200,
			contains: []string{"<!doctype html>", "LibraTrack", `/static/js/app.js`, `id="flash"`, `id="book-list"`, `hx-get="/web/books"`,
				`hx-trigger="keyup changed delay:300ms"`, `hx-target="#book-list"`, "Go in Practice", "Available (2/3)", "Unavailable"},
			excludes: []string{"hx-delete"},
		},
		{
			name: "htmx search returns only row fragments", target: "/web/books?q=Go", headers: htmx, status: 200,
			contains: []string{"<tr", "Go in Practice"},
			excludes: []string{"<html", "<!doctype", "<table", "Python Basics"},
		},
		{
			name: "htmx search without matches returns empty-state row", target: "/web/books?q=zzz", headers: htmx, status: 200,
			contains: []string{"No books found."},
			excludes: []string{"<html", "Go in Practice"},
		},
		{
			name: "librarian sees delete button with htmx attributes", target: "/web/books", headers: htmx, role: model.UserLibrarian, status: 200,
			contains: []string{`hx-delete="/web/books/1"`, `hx-target="closest tr"`, `hx-swap="outerHTML"`, `hx-confirm="Delete this book?"`},
		},
		{
			name: "reader does not see delete button", target: "/web/books", headers: htmx, role: model.UserReader, status: 200,
			contains: []string{"Go in Practice"}, excludes: []string{"hx-delete"},
		},
		{
			name: "titles are contextually autoescaped", target: "/web/books", headers: htmx, status: 200,
			contains: []string{"&lt;script&gt;alert(1)&lt;/script&gt;"}, excludes: []string{"<script>alert(1)"},
		},
		{
			name: "non-htmx fragment request redirects to full page", target: "/web/books?q=Go", status: 303,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newRouter(t, seedBooks(), fakeUsers{})
			cookie := ""
			if tt.role != "" {
				cookie = tokenFor(t, tt.role)
			}
			rec := do(router, http.MethodGet, tt.target, tt.headers, cookie)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.status, rec.Body)
			}
			body := rec.Body.String()
			for _, want := range tt.contains {
				if !strings.Contains(body, want) {
					t.Errorf("body missing %q:\n%s", want, body)
				}
			}
			for _, bad := range tt.excludes {
				if strings.Contains(body, bad) {
					t.Errorf("body unexpectedly contains %q:\n%s", bad, body)
				}
			}
		})
	}
}

func TestCatalogSetsSecurityHeaders(t *testing.T) {
	rec := do(newRouter(t, seedBooks(), fakeUsers{}), http.MethodGet, "/catalog", nil, "")
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("CSP = %q", csp)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing nosniff")
	}
}

func TestDeleteBookRBAC(t *testing.T) {
	htmx := map[string]string{"HX-Request": "true"}
	tests := []struct {
		name        string
		target      string
		headers     map[string]string
		cookie      string
		status      int
		wantDeleted bool
	}{
		{name: "anonymous gets 403", target: "/web/books/1", headers: htmx, status: 403},
		{name: "reader gets 403", target: "/web/books/1", headers: htmx, cookie: "reader", status: 403},
		{name: "invalid token gets 403", target: "/web/books/1", headers: htmx, cookie: "garbage", status: 403},
		{name: "librarian without HX-Request gets 403", target: "/web/books/1", cookie: "librarian", status: 403},
		{name: "librarian deletes with empty 200", target: "/web/books/1", headers: htmx, cookie: "librarian", status: 200, wantDeleted: true},
		{name: "admin deletes with empty 200", target: "/web/books/2", headers: htmx, cookie: "admin", status: 200, wantDeleted: true},
		{name: "unknown id returns 404", target: "/web/books/99", headers: htmx, cookie: "librarian", status: 404},
		{name: "bad id returns 400", target: "/web/books/abc", headers: htmx, cookie: "librarian", status: 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			books := seedBooks()
			router := newRouter(t, books, fakeUsers{})
			cookie := tt.cookie
			switch cookie {
			case "reader", "librarian", "admin":
				cookie = tokenFor(t, cookie)
			}
			rec := do(router, http.MethodDelete, tt.target, tt.headers, cookie)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if tt.status == 200 && rec.Body.Len() != 0 {
				t.Errorf("delete response body must be empty, got %q", rec.Body)
			}
			if got := len(books.deleted) > 0; got != tt.wantDeleted {
				t.Errorf("deleted = %v, want %v", got, tt.wantDeleted)
			}
		})
	}
}

func TestLoginSetsHttpOnlyCookie(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret-pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := &model.User{ID: 5, Email: "lib@example.com", PasswordHash: string(hash), Role: model.UserLibrarian}
	tests := []struct {
		name       string
		password   string
		status     int
		wantCookie bool
	}{
		{name: "valid credentials redirect with cookie", password: "secret-pass", status: 303, wantCookie: true},
		{name: "wrong password re-renders login with 401", password: "nope", status: 401},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newRouter(t, seedBooks(), fakeUsers{user: user})
			form := url.Values{"email": {"lib@example.com"}, "password": {tt.password}}
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			var found *http.Cookie
			for _, c := range rec.Result().Cookies() {
				if c.Name == webhandler.CookieName {
					found = c
				}
			}
			if tt.wantCookie != (found != nil) {
				t.Fatalf("cookie present = %v, want %v", found != nil, tt.wantCookie)
			}
			if found != nil {
				if !found.HttpOnly || found.SameSite != http.SameSiteLaxMode || found.Path != "/" {
					t.Errorf("cookie flags wrong: %+v", found)
				}
				if strings.Contains(rec.Body.String(), found.Value) {
					t.Error("JWT must never appear in the response body")
				}
			}
		})
	}
}
