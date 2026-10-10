package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"homework/internal/repository"
)

func writeRepositoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeLibraryError(c, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, repository.ErrConflict):
		writeLibraryError(c, http.StatusConflict, "conflict", "resource already exists")
	case errors.Is(err, repository.ErrInvalidReference):
		writeLibraryError(c, http.StatusUnprocessableEntity, "invalid_reference", "a referenced resource does not exist")
	default:
		writeLibraryError(c, http.StatusInternalServerError, "internal_server_error", "an internal error occurred")
	}
}

func writeLibraryError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
