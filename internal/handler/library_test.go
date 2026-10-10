package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"homework/internal/model"
	"homework/internal/repository"
)

type fakeBookRepository struct {
	book   *model.Book
	filter repository.BookFilter
	err    error
}

func (r *fakeBookRepository) Create(_ context.Context, book *model.Book) error {
	if r.err != nil {
		return r.err
	}
	book.ID = 7
	r.book = book
	return nil
}

func (r *fakeBookRepository) FindByID(_ context.Context, id uint64) (*model.Book, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.book == nil || r.book.ID != id {
		return nil, repository.ErrNotFound
	}
	return r.book, nil
}

func (r *fakeBookRepository) Update(_ context.Context, book *model.Book) error {
	r.book = book
	return r.err
}

func (r *fakeBookRepository) Delete(_ context.Context, _ uint64) error { return r.err }

func (r *fakeBookRepository) List(_ context.Context, filter repository.BookFilter) ([]model.Book, error) {
	r.filter = filter
	if r.err != nil {
		return nil, r.err
	}
	return []model.Book{}, nil
}

func newLibraryTestRouter(books repository.BookRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewLibraryHandler(books, nil, nil, nil).RegisterRoutes(router.Group("/api/v1"))
	return router
}

func TestLibraryListBooksReturnsEmptyArray(t *testing.T) {
	router := newLibraryTestRouter(&fakeBookRepository{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/books", nil))
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != "[]" {
		t.Fatalf("GET books = %d %q; want 200 []", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("Content-Type = %q; want JSON", response.Header().Get("Content-Type"))
	}
}

func TestLibraryListBooksRejectsInvalidPagination(t *testing.T) {
	router := newLibraryTestRouter(&fakeBookRepository{})
	for _, query := range []string{"?page=0", "?limit=-1", "?page=not-a-number"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/books"+query, nil))
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "invalid_pagination") {
			t.Errorf("GET books%s = %d %s; want 400 invalid_pagination", query, response.Code, response.Body.String())
		}
	}
}

func TestLibraryCreateBookValidatesRequiredRelations(t *testing.T) {
	router := newLibraryTestRouter(&fakeBookRepository{})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/books", strings.NewReader(`{"title":"Title","isbn":"123"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "validation_error") {
		t.Fatalf("POST books = %d %s; want 422 validation_error", response.Code, response.Body.String())
	}
}

func TestLibraryBookListMapsRepositoryErrorsSafely(t *testing.T) {
	fake := &fakeBookRepository{err: errors.New("database secret")}
	router := newLibraryTestRouter(fake)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/books", nil))
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "database secret") {
		t.Fatalf("GET books leaked error or returned wrong status: %d %s", response.Code, response.Body.String())
	}
}
