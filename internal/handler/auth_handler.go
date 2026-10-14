package handler

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"homework/internal/middleware"
	"homework/internal/model"
	"homework/internal/repository"
)

type authUserStore interface {
	Create(context.Context, *model.User) error
	FindByEmail(context.Context, string) (*model.User, error)
}

type AuthHandler struct{ users authUserStore }

func NewAuthHandler(users authUserStore) *AuthHandler { return &AuthHandler{users: users} }

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	Email string `json:"email"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input RegisterRequest
	if !bindJSON(c, &input) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	parsedEmail, emailErr := mail.ParseAddress(email)
	if emailErr != nil || parsedEmail.Address != email || len([]byte(input.Password)) < 8 || len([]byte(input.Password)) > 72 {
		writeLibraryError(c, http.StatusUnprocessableEntity, "validation_error", "a valid email and password of 8 to 72 bytes are required")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		writeLibraryError(c, http.StatusInternalServerError, "internal_server_error", "an internal error occurred")
		return
	}
	user := &model.User{Email: email, PasswordHash: string(hash), Role: model.UserReader}
	if err := h.users.Create(c.Request.Context(), user); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			writeLibraryError(c, http.StatusUnprocessableEntity, "duplicate_email", "email is already registered")
			return
		}
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input LoginRequest
	if !bindJSON(c, &input) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	user, err := h.users.FindByEmail(c.Request.Context(), email)
	if err != nil || user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		writeLibraryError(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
		return
	}
	secret := os.Getenv("JWT_SECRET")
	token, err := middleware.GenerateToken(*user, secret)
	if err != nil {
		writeLibraryError(c, http.StatusInternalServerError, "configuration_error", "authentication service is not configured")
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": token, "token_type": "Bearer", "expires_in": 86400})
}
