package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"homework/internal/model"
	"homework/internal/repository"
)

type LibraryHandler struct {
	books      repository.BookRepository
	authors    *repository.AuthorRepository
	categories *repository.CategoryRepository
	copies     *repository.CopyRepository
}

func NewLibraryHandler(books repository.BookRepository, authors *repository.AuthorRepository, categories *repository.CategoryRepository, copies *repository.CopyRepository) *LibraryHandler {
	return &LibraryHandler{books: books, authors: authors, categories: categories, copies: copies}
}

func (h *LibraryHandler) RegisterPublicRoutes(router gin.IRouter) {
	router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	router.GET("/categories", h.listCategories)
	router.GET("/categories/:id", h.getCategory)
	router.GET("/authors", h.listAuthors)
	router.GET("/authors/:id", h.getAuthor)
	router.GET("/books", h.listBooks)
	router.GET("/books/:id", h.getBook)
	router.GET("/books/:id/copies", h.listCopies)
	router.GET("/books/:id/availability", h.availability)
}

func (h *LibraryHandler) RegisterStaffRoutes(router gin.IRouter) {
	router.POST("/categories", h.createCategory)
	router.POST("/authors", h.createAuthor)
	router.POST("/books", h.createBook)
	router.PUT("/books/:id", h.updateBook)
	router.DELETE("/books/:id", h.deleteBook)
	router.POST("/books/:id/copies", h.createCopy)
}

// RegisterRoutes preserves the legacy ungrouped registration for handler tests.
func (h *LibraryHandler) RegisterRoutes(router gin.IRouter) {
	h.RegisterPublicRoutes(router)
	h.RegisterStaffRoutes(router)
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		writeLibraryError(c, http.StatusUnprocessableEntity, "validation_error", "request body must be a valid JSON object")
		return false
	}
	return true
}

func validBookInput(input bookInput) bool {
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.ISBN) == "" || input.CategoryID == 0 || len(input.AuthorIDs) == 0 {
		return false
	}
	for _, id := range input.AuthorIDs {
		if id == 0 {
			return false
		}
	}
	return true
}

func authorRefs(ids []uint64) []model.Author {
	authors := make([]model.Author, 0, len(ids))
	for _, id := range ids {
		authors = append(authors, model.Author{ID: id})
	}
	return authors
}

func pathID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		writeLibraryError(c, http.StatusBadRequest, "invalid_id", "id must be a positive integer")
		return 0, false
	}
	return id, true
}

func queryPositiveInt(c *gin.Context, name string, fallback int) (int, error) {
	value := c.Query(name)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, errors.New("invalid positive integer")
	}
	return parsed, nil
}

func queryOptionalUint(c *gin.Context, name string) (uint64, error) {
	value := c.Query(name)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, errors.New("invalid positive integer")
	}
	return parsed, nil
}
