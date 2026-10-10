package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"homework/internal/model"
	"homework/internal/repository"
)

type bookInput struct {
	Title         string   `json:"title"`
	ISBN          string   `json:"isbn"`
	CategoryID    uint64   `json:"category_id"`
	AuthorIDs     []uint64 `json:"author_ids"`
	PublishedYear *int     `json:"published_year"`
	Description   *string  `json:"description"`
}

func (h *LibraryHandler) createBook(c *gin.Context) {
	var input bookInput
	if !bindJSON(c, &input) {
		return
	}
	if !validBookInput(input) {
		writeLibraryError(c, http.StatusUnprocessableEntity, "validation_error", "title, isbn, category_id, and at least one author_id are required")
		return
	}
	book := model.Book{Title: strings.TrimSpace(input.Title), ISBN: strings.TrimSpace(input.ISBN), CategoryID: input.CategoryID, PublishedYear: input.PublishedYear, Description: input.Description, Authors: authorRefs(input.AuthorIDs)}
	if err := h.books.Create(c.Request.Context(), &book); err != nil {
		writeRepositoryError(c, err)
		return
	}
	created, err := h.books.FindByID(c.Request.Context(), book.ID)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *LibraryHandler) listBooks(c *gin.Context) {
	page, err := queryPositiveInt(c, "page", 1)
	if err != nil {
		writeLibraryError(c, http.StatusBadRequest, "invalid_pagination", "page must be a positive integer")
		return
	}
	limit, err := queryPositiveInt(c, "limit", 10)
	if err != nil {
		writeLibraryError(c, http.StatusBadRequest, "invalid_pagination", "limit must be a positive integer")
		return
	}
	categoryID, err := queryOptionalUint(c, "category_id")
	if err != nil {
		writeLibraryError(c, http.StatusBadRequest, "invalid_pagination", "category_id must be a positive integer")
		return
	}
	books, err := h.books.List(c.Request.Context(), repository.BookFilter{CategoryID: categoryID, Page: page, Limit: limit})
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, books)
}

func (h *LibraryHandler) getBook(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	book, err := h.books.FindByID(c.Request.Context(), id)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, book)
}

func (h *LibraryHandler) updateBook(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var input bookInput
	if !bindJSON(c, &input) {
		return
	}
	if !validBookInput(input) {
		writeLibraryError(c, http.StatusUnprocessableEntity, "validation_error", "title, isbn, category_id, and at least one author_id are required")
		return
	}
	book := &model.Book{ID: id, Title: strings.TrimSpace(input.Title), ISBN: strings.TrimSpace(input.ISBN), CategoryID: input.CategoryID, PublishedYear: input.PublishedYear, Description: input.Description, Authors: authorRefs(input.AuthorIDs)}
	if err := h.books.Update(c.Request.Context(), book); err != nil {
		writeRepositoryError(c, err)
		return
	}
	updated, err := h.books.FindByID(c.Request.Context(), id)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *LibraryHandler) deleteBook(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.books.Delete(c.Request.Context(), id); err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
