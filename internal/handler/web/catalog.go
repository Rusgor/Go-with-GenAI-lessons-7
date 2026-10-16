// Package web serves the server-rendered LibraTrack UI and its HTMX endpoints.
package web

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"homework/internal/middleware"
	"homework/internal/model"
	"homework/internal/repository"
)

const (
	// CookieName is the HttpOnly cookie that carries the JWT for browser sessions.
	CookieName = "libratrack_token"

	catalogLimit  = 100
	maxQueryRunes = 100
	tokenTTL      = 24 * time.Hour
)

type BookStore interface {
	List(context.Context, repository.BookFilter) ([]model.Book, error)
	Delete(context.Context, uint64) error
}

type AvailabilityStore interface {
	AvailabilityByBooks(context.Context, []uint64) (map[uint64]repository.BookAvailability, error)
}

type UserStore interface {
	FindByEmail(context.Context, string) (*model.User, error)
}

type Handler struct {
	books  BookStore
	copies AvailabilityStore
	users  UserStore
	pages  map[string]*template.Template
}

func New(books BookStore, copies AvailabilityStore, users UserStore, pages map[string]*template.Template) *Handler {
	return &Handler{books: books, copies: copies, users: users, pages: pages}
}

// bookView is the template model; it carries only display strings and flags.
type bookView struct {
	ID        uint64
	Title     string
	Authors   string
	Category  string
	ISBN      string
	Total     int64
	Available int64
	CanDelete bool
}

type pageData struct {
	Title     string
	Query     string
	Books     []bookView
	Role      string
	LoggedIn  bool
	CanDelete bool
	Error     string
}

// Register mounts the UI routes. Static assets are mounted by the caller.
func (h *Handler) Register(router gin.IRouter) {
	web := router.Group("")
	web.Use(securityHeaders(), middleware.OptionalCookieAuth(CookieName))
	web.GET("/", func(c *gin.Context) { c.Redirect(http.StatusSeeOther, "/catalog") })
	web.GET("/catalog", h.catalog)
	web.GET("/login", h.loginPage)
	web.POST("/login", h.login)
	web.POST("/logout", h.logout)
	web.GET("/web/books", h.bookRows)
	web.DELETE("/web/books/:id", requireHTMXStaff(), h.deleteBook)
}

// securityHeaders restricts the page to same-origin resources; htmx needs no inline script or eval.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "same-origin")
		c.Next()
	}
}

// requireHTMXStaff protects state-changing HTMX endpoints. Anonymous callers and readers
// get 403. The HX-Request header is a CSRF defence in depth: browsers cannot attach it
// to a cross-site form post, and cross-origin fetches need a CORS preflight that is refused.
func requireHTMXStaff() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := currentRole(c)
		if (role != model.UserLibrarian && role != model.UserAdmin) || c.GetHeader("HX-Request") != "true" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

func currentRole(c *gin.Context) string {
	role, _ := c.Get(middleware.ContextRole)
	value, _ := role.(string)
	return value
}

func canDelete(role string) bool { return role == model.UserLibrarian || role == model.UserAdmin }

func (h *Handler) catalog(c *gin.Context) {
	role := currentRole(c)
	query := cleanQuery(c.Query("q"))
	books, err := h.loadBooks(c.Request.Context(), query, role)
	if err != nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	h.renderPage(c, http.StatusOK, "catalog", pageData{
		Title: "Catalog", Query: query, Books: books, Role: role, LoggedIn: role != "", CanDelete: canDelete(role),
	})
}

// bookRows answers the live-search request with only the <tr> fragments (no layout).
func (h *Handler) bookRows(c *gin.Context) {
	query := cleanQuery(c.Query("q"))
	if c.GetHeader("HX-Request") != "true" {
		target := "/catalog"
		if query != "" {
			target += "?q=" + url.QueryEscape(query)
		}
		c.Redirect(http.StatusSeeOther, target)
		return
	}
	books, err := h.loadBooks(c.Request.Context(), query, currentRole(c))
	if err != nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	var buf bytes.Buffer
	if err := h.pages["catalog"].ExecuteTemplate(&buf, "book_rows", books); err != nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

// deleteBook removes a book and answers with an empty 200 so htmx drops the targeted row.
func (h *Handler) deleteBook(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.String(http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.books.Delete(c.Request.Context(), id); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			c.String(http.StatusNotFound, "book not found")
		case errors.Is(err, repository.ErrInvalidReference):
			c.String(http.StatusConflict, "book has loan history and cannot be deleted")
		default:
			c.String(http.StatusInternalServerError, "internal error")
		}
		return
	}
	c.Status(http.StatusOK)
	c.Writer.WriteHeaderNow()
}

func (h *Handler) loginPage(c *gin.Context) {
	h.renderPage(c, http.StatusOK, "login", pageData{Title: "Log in"})
}

func (h *Handler) login(c *gin.Context) {
	email := strings.ToLower(strings.TrimSpace(c.PostForm("email")))
	password := c.PostForm("password")
	user, err := h.users.FindByEmail(c.Request.Context(), email)
	if err != nil || user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		h.renderPage(c, http.StatusUnauthorized, "login", pageData{Title: "Log in", Error: "Invalid credentials"})
		return
	}
	token, err := middleware.GenerateToken(*user, os.Getenv("JWT_SECRET"))
	if err != nil {
		c.String(http.StatusInternalServerError, "authentication service is not configured")
		return
	}
	setTokenCookie(c, token, int(tokenTTL.Seconds()))
	c.Redirect(http.StatusSeeOther, "/catalog")
}

func (h *Handler) logout(c *gin.Context) {
	setTokenCookie(c, "", -1)
	c.Redirect(http.StatusSeeOther, "/login")
}

// setTokenCookie stores the JWT where page scripts cannot read it: HttpOnly blocks
// document.cookie access (XSS), SameSite=Lax withholds it from cross-site POST/DELETE.
func setTokenCookie(c *gin.Context, value string, maxAge int) {
	secure := c.Request.TLS != nil || os.Getenv("COOKIE_SECURE") == "true"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieName, value, maxAge, "/", "", secure, true)
}

func (h *Handler) loadBooks(ctx context.Context, query, role string) ([]bookView, error) {
	books, err := h.books.List(ctx, repository.BookFilter{Query: query, Page: 1, Limit: catalogLimit})
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(books))
	for _, book := range books {
		ids = append(ids, book.ID)
	}
	availability, err := h.copies.AvailabilityByBooks(ctx, ids)
	if err != nil {
		return nil, err
	}
	views := make([]bookView, 0, len(books))
	for _, book := range books {
		names := make([]string, 0, len(book.Authors))
		for _, author := range book.Authors {
			names = append(names, author.FullName)
		}
		a := availability[book.ID]
		views = append(views, bookView{
			ID: book.ID, Title: book.Title, Authors: strings.Join(names, ", "), Category: book.Category.Name,
			ISBN: book.ISBN, Total: a.Total, Available: a.Available, CanDelete: canDelete(role),
		})
	}
	return views, nil
}

func (h *Handler) renderPage(c *gin.Context, status int, page string, data pageData) {
	var buf bytes.Buffer
	if err := h.pages[page].ExecuteTemplate(&buf, "layout.html", data); err != nil {
		c.String(http.StatusInternalServerError, "internal error")
		return
	}
	c.Data(status, "text/html; charset=utf-8", buf.Bytes())
}

func cleanQuery(q string) string {
	q = strings.TrimSpace(q)
	if r := []rune(q); len(r) > maxQueryRunes {
		q = string(r[:maxQueryRunes])
	}
	return q
}
