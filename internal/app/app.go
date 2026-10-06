// Package app wires the whole HTTP API together.
//
// The automatic tests talk to your API ONLY through NewRouter(), so you are free to
// organise the rest of the code (model, repository, handler, middleware) as you like.
package app

import (
	"net/http"

	"homework/internal/handler"
	"homework/internal/middleware"
	"homework/internal/repository"
)

// NewRouter must return a fully configured HTTP handler for your resource.
//
// Requirements (see README.md for the full contract):
//   - every call returns a NEW router with its own EMPTY in-memory storage;
//   - it must be safe for concurrent requests;
//   - it can be built with net/http, gin, chi or any other router.
func NewRouter() http.Handler {
	store := repository.NewMemoryDeviceStore()
	devices := handler.NewDeviceHandler(store)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", devices.Health)
	mux.HandleFunc("/api/v1/devices", devices.Collection)
	mux.HandleFunc("/api/v1/devices/", devices.Item)
	mux.HandleFunc("/", handler.NotFound)
	return middleware.CORS(mux)
}
