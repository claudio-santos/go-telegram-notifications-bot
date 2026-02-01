package internal

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router sets up the application routes.
func Router(h *Handlers) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Static files
	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))).ServeHTTP(w, r)
	})

	r.Get("/", h.IndexGetHandler)
	r.Post("/", h.IndexPostHandler)
	r.Get("/config", h.ConfigGetHandler)
	r.Post("/config", h.ConfigPostHandler)

	return r
}
