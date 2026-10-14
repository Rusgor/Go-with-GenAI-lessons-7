package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"homework/internal/middleware"
	"homework/internal/model"
	"homework/internal/repository"
)

const securityTestSecret = "unit-test-secret-with-sufficient-entropy"

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

func securityRequest(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
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

func securityToken(t *testing.T, userID uint64, role string) string {
	t.Helper()
	token, err := middleware.GenerateToken(model.User{ID: userID, Role: role}, securityTestSecret)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func expiredSecurityToken(t *testing.T) string {
	t.Helper()
	claims := middleware.TokenClaims{
		UserID: 1,
		Role:   model.UserAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(securityTestSecret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}
