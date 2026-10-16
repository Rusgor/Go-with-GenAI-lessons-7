package app

import (
	"html/template"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"homework/internal/handler"
	webhandler "homework/internal/handler/web"
	"homework/internal/middleware"
	"homework/internal/model"
	"homework/internal/repository"
)

// Option customises NewLibraryRouter.
type Option func(*routerOptions)

type routerOptions struct {
	pages  map[string]*template.Template
	static fs.FS
}

// WithWeb mounts the server-rendered UI (templates) and its static assets.
func WithWeb(pages map[string]*template.Template, static fs.FS) Option {
	return func(o *routerOptions) { o.pages, o.static = pages, static }
}

// NewLibraryRouter builds the LibraTrack API without changing the devices router.
func NewLibraryRouter(db *gorm.DB, opts ...Option) http.Handler {
	var options routerOptions
	for _, opt := range opts {
		opt(&options)
	}
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), libraryCORS())
	router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	router.GET("/swagger/index.html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<!doctype html><html><body><h1>LibraTrack API</h1><p>OpenAPI documentation is available from the API routes.</p></body></html>"))
	})

	api := router.Group("/api/v1")
	users := repository.NewUserRepository(db)
	loans := repository.NewLoanRepository(db)
	userHandler := handler.NewUserHandler(users)
	loanHandler := handler.NewLoanHandler(loans)
	library := handler.NewLibraryHandler(
		repository.NewGormBookRepository(db),
		repository.NewAuthorRepository(db),
		repository.NewCategoryRepository(db),
		repository.NewCopyRepository(db),
	)
	public := api.Group("")
	library.RegisterPublicRoutes(public)
	auth := handler.NewAuthHandler(users)
	public.POST("/auth/register", auth.Register)
	public.POST("/auth/login", auth.Login)

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	protected.GET("/me/loans", loanHandler.MyLoans)

	staff := protected.Group("")
	staff.Use(middleware.RequireRole(model.UserLibrarian, model.UserAdmin))
	library.RegisterStaffRoutes(staff)
	staff.POST("/loans", loanHandler.CreateLoan)
	staff.POST("/loans/:id/return", loanHandler.ReturnLoan)

	admin := protected.Group("")
	admin.Use(middleware.RequireRole(model.UserAdmin))
	admin.GET("/users", userHandler.ListUsers)
	admin.PUT("/users/:id/role", userHandler.UpdateUserRole)

	if options.pages != nil {
		webHandler := webhandler.New(
			repository.NewGormBookRepository(db),
			repository.NewCopyRepository(db),
			users,
			options.pages,
		)
		webHandler.Register(router)
	}
	if options.static != nil {
		router.StaticFS("/static", http.FS(options.static))
	}
	return router
}

func libraryCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
