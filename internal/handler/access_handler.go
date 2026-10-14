package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"homework/internal/middleware"
	"homework/internal/model"
)

type userAdminStore interface {
	List(context.Context) ([]model.User, error)
	UpdateRole(context.Context, uint64, string) error
}

type loanStore interface {
	Create(context.Context, uint64, uint64, time.Time) (*model.Loan, error)
	ListByUser(context.Context, uint64) ([]model.Loan, error)
	Return(context.Context, uint64) (*model.Loan, error)
}

type AccessHandler struct {
	users userAdminStore
	loans loanStore
}

func NewAccessHandler(users userAdminStore, loans loanStore) *AccessHandler {
	return &AccessHandler{users: users, loans: loans}
}

func (h *AccessHandler) ListUsers(c *gin.Context) {
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

func (h *AccessHandler) UpdateUserRole(c *gin.Context) {
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

type createLoanRequest struct {
	UserID uint64    `json:"user_id"`
	CopyID uint64    `json:"copy_id"`
	DueAt  time.Time `json:"due_at"`
}

func (h *AccessHandler) CreateLoan(c *gin.Context) {
	var input createLoanRequest
	if !bindJSON(c, &input) {
		return
	}
	if input.UserID == 0 || input.CopyID == 0 || input.DueAt.IsZero() || !input.DueAt.After(time.Now()) {
		writeLibraryError(c, http.StatusUnprocessableEntity, "validation_error", "user_id, copy_id, and a future due_at are required")
		return
	}
	loan, err := h.loans.Create(c.Request.Context(), input.UserID, input.CopyID, input.DueAt)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusCreated, loan)
}

func (h *AccessHandler) MyLoans(c *gin.Context) {
	userID, ok := c.Get(middleware.ContextUserID)
	id, valid := userID.(uint64)
	if !ok || !valid || id == 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "authentication required"}})
		return
	}
	loans, err := h.loans.ListByUser(c.Request.Context(), id)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, loans)
}

func (h *AccessHandler) ReturnLoan(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		writeLibraryError(c, http.StatusBadRequest, "invalid_id", "id must be a positive integer")
		return
	}
	loan, err := h.loans.Return(c.Request.Context(), id)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, loan)
}
