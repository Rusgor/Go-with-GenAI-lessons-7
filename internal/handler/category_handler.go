package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"homework/internal/model"
)

type categoryInput struct {
	Name string `json:"name"`
}

func (h *LibraryHandler) createCategory(c *gin.Context) {
	var input categoryInput
	if !bindJSON(c, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		writeLibraryError(c, http.StatusUnprocessableEntity, "validation_error", "name is required")
		return
	}
	category := model.Category{Name: input.Name}
	if err := h.categories.Create(c.Request.Context(), &category); err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusCreated, category)
}

func (h *LibraryHandler) listCategories(c *gin.Context) {
	categories, err := h.categories.List(c.Request.Context())
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, categories)
}

func (h *LibraryHandler) getCategory(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	category, err := h.categories.FindByID(c.Request.Context(), id)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, category)
}
