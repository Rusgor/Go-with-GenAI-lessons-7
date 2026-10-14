package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"homework/internal/handler"
	"homework/internal/middleware"
	"homework/internal/model"
	"homework/internal/repository"
)

// NewLibraryRouter builds the LibraTrack API without changing the devices router.
func NewLibraryRouter(db *gorm.DB) http.Handler {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), libraryCORS())
	router.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	router.GET("/swagger/index.html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<!doctype html><html><body><h1>LibraTrack API</h1><p>OpenAPI documentation is available from the API routes.</p></body></html>"))
	})

	api := router.Group("/api/v1")
	users := repository.NewUserRepository(db)
	loans := repository.NewLoanRepository(db)
	access := handler.NewAccessHandler(users, loans)
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
	protected.GET("/me/loans", access.MyLoans)

	staff := protected.Group("")
	staff.Use(middleware.RequireRole(model.UserLibrarian, model.UserAdmin))
	library.RegisterStaffRoutes(staff)
	staff.POST("/loans", access.CreateLoan)
	staff.POST("/loans/:id/return", access.ReturnLoan)

	admin := protected.Group("")
	admin.Use(middleware.RequireRole(model.UserAdmin))
	admin.GET("/users", access.ListUsers)
	admin.PUT("/users/:id/role", access.UpdateUserRole)
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
