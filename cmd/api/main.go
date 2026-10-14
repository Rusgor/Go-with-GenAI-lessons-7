package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"homework/internal/app"
	"homework/internal/repository"
)

func main() {
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		runLibraryAPI(databaseURL)
		return
	}

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, app.NewRouter()); err != nil {
		log.Fatal(err)
	}
}

func runLibraryAPI(databaseURL string) {
	if len([]byte(os.Getenv("JWT_SECRET"))) < 32 {
		log.Fatal("JWT_SECRET must be configured with at least 32 bytes")
	}
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "migrations"
	}
	if err := repository.ApplyMigrations(databaseURL, migrationsPath); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	db, err := repository.OpenDB(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get database pool: %v", err)
	}
	defer sqlDB.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{Addr: ":" + port, Handler: app.NewLibraryRouter(db), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	log.Printf("LibraTrack API listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve API: %v", err)
	}
}
