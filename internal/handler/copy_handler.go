package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"homework/internal/model"
)

type copyInput struct {
	InventoryNo string `json:"inventory_no"`
	Status      string `json:"status"`
}

func (h *LibraryHandler) createCopy(c *gin.Context) {
	bookID, ok := pathID(c)
	if !ok {
		return
	}
	if _, err := h.books.FindByID(c.Request.Context(), bookID); err != nil {
		writeRepositoryError(c, err)
		return
	}
	var input copyInput
	if !bindJSON(c, &input) {
		return
	}
	input.InventoryNo = strings.TrimSpace(input.InventoryNo)
	if input.InventoryNo == "" {
		writeLibraryError(c, http.StatusUnprocessableEntity, "validation_error", "inventory_no is required")
		return
	}
	if input.Status == "" {
		input.Status = model.CopyAvailable
	}
	if input.Status != model.CopyAvailable && input.Status != model.CopyCheckedOut && input.Status != model.CopyLost {
		writeLibraryError(c, http.StatusUnprocessableEntity, "validation_error", "status must be available, checked_out, or lost")
		return
	}
	copy := model.Copy{BookID: bookID, InventoryNo: input.InventoryNo, Status: input.Status}
	if err := h.copies.Create(c.Request.Context(), &copy); err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusCreated, copy)
}

func (h *LibraryHandler) listCopies(c *gin.Context) {
	bookID, ok := pathID(c)
	if !ok {
		return
	}
	if _, err := h.books.FindByID(c.Request.Context(), bookID); err != nil {
		writeRepositoryError(c, err)
		return
	}
	copies, err := h.copies.ListByBook(c.Request.Context(), bookID)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, copies)
}

func (h *LibraryHandler) availability(c *gin.Context) {
	bookID, ok := pathID(c)
	if !ok {
		return
	}
	total, available, err := h.copies.Availability(c.Request.Context(), bookID)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"book_id": bookID, "total_copies": total, "available_copies": available})
}
