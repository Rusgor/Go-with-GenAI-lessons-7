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

type loanStore interface {
	Create(context.Context, uint64, uint64, time.Time) (*model.Loan, error)
	ListByUser(context.Context, uint64) ([]model.Loan, error)
	Return(context.Context, uint64) (*model.Loan, error)
}

type LoanHandler struct{ loans loanStore }

func NewLoanHandler(loans loanStore) *LoanHandler { return &LoanHandler{loans: loans} }

type createLoanRequest struct {
	UserID uint64    `json:"user_id"`
	CopyID uint64    `json:"copy_id"`
	DueAt  time.Time `json:"due_at"`
}

func (h *LoanHandler) CreateLoan(c *gin.Context) {
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

func (h *LoanHandler) MyLoans(c *gin.Context) {
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

func (h *LoanHandler) ReturnLoan(c *gin.Context) {
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
