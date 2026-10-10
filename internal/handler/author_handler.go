package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"homework/internal/model"
)

type authorInput struct {
	FullName string  `json:"full_name"`
	Bio      *string `json:"bio"`
}

func (h *LibraryHandler) createAuthor(c *gin.Context) {
	var input authorInput
	if !bindJSON(c, &input) {
		return
	}
	input.FullName = strings.TrimSpace(input.FullName)
	if input.FullName == "" {
		writeLibraryError(c, http.StatusUnprocessableEntity, "validation_error", "full_name is required")
		return
	}
	author := model.Author{FullName: input.FullName, Bio: input.Bio}
	if err := h.authors.Create(c.Request.Context(), &author); err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusCreated, author)
}

func (h *LibraryHandler) listAuthors(c *gin.Context) {
	authors, err := h.authors.List(c.Request.Context())
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, authors)
}

func (h *LibraryHandler) getAuthor(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	author, err := h.authors.FindByID(c.Request.Context(), id)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, author)
}
