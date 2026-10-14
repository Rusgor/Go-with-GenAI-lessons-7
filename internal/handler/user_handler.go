package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"homework/internal/model"
)

type userAdminStore interface {
	List(context.Context) ([]model.User, error)
	UpdateRole(context.Context, uint64, string) error
}

type UserHandler struct{ users userAdminStore }

func NewUserHandler(users userAdminStore) *UserHandler { return &UserHandler{users: users} }

func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.users.List(c.Request.Context())
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, users)
}

type updateRoleRequest struct {
	Role string `json:"role"`
}

func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		writeLibraryError(c, http.StatusBadRequest, "invalid_id", "id must be a positive integer")
		return
	}
	var input updateRoleRequest
	if !bindJSON(c, &input) {
		return
	}
	if input.Role != model.UserReader && input.Role != model.UserLibrarian && input.Role != model.UserAdmin {
		writeLibraryError(c, http.StatusUnprocessableEntity, "validation_error", "role must be reader, librarian, or admin")
		return
	}
	if err := h.users.UpdateRole(c.Request.Context(), id, input.Role); err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
