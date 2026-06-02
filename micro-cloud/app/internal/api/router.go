package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func SetupRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", h.HealthCheck)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/tenants", func(r chi.Router) {
			r.Post("/", h.CreateTenant)
			r.Get("/", h.ListTenants)
			r.Get("/{id}", h.GetTenant)
		})

		r.Route("/volumes", func(r chi.Router) {
			r.Post("/", h.CreateVolume)
			r.Get("/", h.ListVolumes)
		})

		r.Route("/workspaces", func(r chi.Router) {
			r.Post("/", h.CreateWorkspace)
			r.Get("/", h.ListWorkspaces)
			r.Get("/{id}", h.GetWorkspace)
			r.Delete("/{id}", h.DeleteWorkspace)
		})
	})

	return r
}
