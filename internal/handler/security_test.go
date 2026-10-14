package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"homework/internal/middleware"
	"homework/internal/model"
	"homework/internal/repository"
)

type memoryUsers struct {
	mu    sync.Mutex
	next  uint64
	users map[string]*model.User
}

func newMemoryUsers() *memoryUsers { return &memoryUsers{next: 1, users: map[string]*model.User{}} }

func (m *memoryUsers) Create(_ context.Context, user *model.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[user.Email]; ok {
		return repository.ErrConflict
	}
	copy := *user
	copy.ID = m.next
	m.next++
	m.users[user.Email] = &copy
	*user = copy
	return nil
}

func (m *memoryUsers) FindByEmail(_ context.Context, email string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.users[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copy := *user
	return &copy, nil
}

type memoryBooks struct{ created model.Book }

func (m *memoryBooks) Create(_ context.Context, book *model.Book) error {
	book.ID = 1
	m.created = *book
	return nil
}
func (m *memoryBooks) FindByID(_ context.Context, id uint64) (*model.Book, error) {
	book := m.created
	book.ID = id
	return &book, nil
}
func (m *memoryBooks) Update(context.Context, *model.Book) error { return nil }
func (m *memoryBooks) Delete(context.Context, uint64) error      { return nil }
func (m *memoryBooks) List(context.Context, repository.BookFilter) ([]model.Book, error) {
	return []model.Book{}, nil
}

func securityRouter(users *memoryUsers) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	auth := NewAuthHandler(users)
	api.POST("/auth/register", auth.Register)
	api.POST("/auth/login", auth.Login)
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	staff := protected.Group("")
	staff.Use(middleware.RequireRole(model.UserLibrarian, model.UserAdmin))
	library := NewLibraryHandler(&memoryBooks{}, nil, nil, nil)
	staff.POST("/books", library.createBook)
	protected.GET("/admin-check", middleware.RequireRole(model.UserAdmin), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return router
}

func request(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(response, req)
	return response
}

func TestAuthRegistrationLoginAndEnumerationResistance(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret-with-sufficient-entropy")
	users := newMemoryUsers()
	router := securityRouter(users)

	created := request(router, http.MethodPost, "/api/v1/auth/register", `{"email":"Reader@example.com","password":"correct horse battery staple","role":"admin"}`, "")
	if created.Code != http.StatusCreated {
		t.Fatalf("register = %d, want 201: %s", created.Code, created.Body.String())
	}
	if strings.Contains(created.Body.String(), "password_hash") || !strings.Contains(created.Body.String(), `"role":"reader"`) {
		t.Fatalf("registration leaked hash or accepted caller role: %s", created.Body.String())
	}
	duplicate := request(router, http.MethodPost, "/api/v1/auth/register", `{"email":"reader@example.com","password":"correct horse battery staple"}`, "")
	if duplicate.Code != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate registration = %d, want 422", duplicate.Code)
	}

	login := request(router, http.MethodPost, "/api/v1/auth/login", `{"email":"reader@example.com","password":"correct horse battery staple"}`, "")
	if login.Code != http.StatusOK {
		t.Fatalf("login = %d, want 200: %s", login.Code, login.Body.String())
	}
	var loginBody struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &loginBody); err != nil || loginBody.AccessToken == "" {
		t.Fatalf("login did not return token: %s", login.Body.String())
	}
	parsed, err := jwt.Parse(loginBody.AccessToken, func(token *jwt.Token) (any, error) { return []byte("unit-test-secret-with-sufficient-entropy"), nil })
	if err != nil || !parsed.Valid || parsed.Method.Alg() != jwt.SigningMethodHS256.Alg() {
		t.Fatalf("issued token invalid or not HS256: %v", err)
	}

	wrongPassword := request(router, http.MethodPost, "/api/v1/auth/login", `{"email":"reader@example.com","password":"not-the-password"}`, "")
	unknownEmail := request(router, http.MethodPost, "/api/v1/auth/login", `{"email":"missing@example.com","password":"not-the-password"}`, "")
	if wrongPassword.Code != http.StatusUnauthorized || unknownEmail.Code != http.StatusUnauthorized || wrongPassword.Body.String() != unknownEmail.Body.String() {
		t.Fatalf("login enumeration response differs: wrong=%d %s unknown=%d %s", wrongPassword.Code, wrongPassword.Body.String(), unknownEmail.Code, unknownEmail.Body.String())
	}
}

func TestRBACBookCreationAndTokenValidation(t *testing.T) {
	t.Setenv("JWT_SECRET", "unit-test-secret-with-sufficient-entropy")
	users := newMemoryUsers()
	router := securityRouter(users)
	payload := `{"title":"Safe Book","isbn":"9780000000999","category_id":1,"author_ids":[1]}`

	missing := request(router, http.MethodPost, "/api/v1/books", payload, "")
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing token = %d, want 401", missing.Code)
	}
	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.TokenClaims{UserID: 1, Role: model.UserAdmin, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}})
	expiredToken, err := expired.SignedString([]byte("unit-test-secret-with-sufficient-entropy"))
	if err != nil {
		t.Fatal(err)
	}
	badToken := request(router, http.MethodPost, "/api/v1/books", payload, expiredToken)
	if badToken.Code != http.StatusUnauthorized {
		t.Fatalf("expired token = %d, want 401", badToken.Code)
	}

	for _, role := range []string{model.UserReader, model.UserLibrarian, model.UserAdmin} {
		t.Run(role, func(t *testing.T) {
			hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.DefaultCost)
			if err != nil {
				t.Fatal(err)
			}
			user := model.User{ID: 10, Email: role + "@example.com", PasswordHash: string(hash), Role: role}
			if err := users.Create(context.Background(), &user); err != nil && !errors.Is(err, repository.ErrConflict) {
				t.Fatal(err)
			}
			token, err := middleware.GenerateToken(user, "unit-test-secret-with-sufficient-entropy")
			if err != nil {
				t.Fatal(err)
			}
			response := request(router, http.MethodPost, "/api/v1/books", payload, token)
			want := http.StatusForbidden
			if role == model.UserLibrarian || role == model.UserAdmin {
				want = http.StatusCreated
			}
			if response.Code != want {
				t.Fatalf("book creation for %s = %d, want %d: %s", role, response.Code, want, response.Body.String())
			}
		})
	}

	reader := model.User{ID: 42, Role: model.UserReader}
	readerToken, err := middleware.GenerateToken(reader, "unit-test-secret-with-sufficient-entropy")
	if err != nil {
		t.Fatal(err)
	}
	adminOnly := request(router, http.MethodGet, "/api/v1/admin-check", "", readerToken)
	if adminOnly.Code != http.StatusForbidden {
		t.Fatalf("reader admin route = %d, want 403", adminOnly.Code)
	}
}
