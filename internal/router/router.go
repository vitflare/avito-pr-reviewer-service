package router

import (
	"net/http"
	"time"

	middleware2 "pr-reviewer-service/pkg/middleware"

	"pr-reviewer-service/internal/handler"
	"pr-reviewer-service/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

func SetupRouter(
	authHandler *handler.AuthHandler,
	teamHandler *handler.TeamHandler,
	userHandler *handler.UserHandler,
	prHandler *handler.PRHandler,
	healthHandler *handler.HealthHandler,
	statisticsHandler *handler.StatisticsHandler,
	authService middleware.AuthService,
) http.Handler {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware2.LoggingMiddleware)
	r.Use(chimiddleware.Timeout(300 * time.Millisecond)) // 300ms SLI

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// Public endpoints
	r.Head("/health", healthHandler.Health)
	r.Post("/auth/login", authHandler.Login)

	// Protected endpoints (require JWT authentication)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(authService))

		// Team endpoints
		r.Post("/team/add", teamHandler.CreateTeam)
		r.Get("/team/get", teamHandler.GetTeam)

		// User endpoints
		r.Get("/users/getReview", userHandler.GetReview)

		// Pull Request endpoints
		r.Post("/pullRequest/create", prHandler.CreatePR)
		r.Post("/pullRequest/merge", prHandler.MergePR)
		r.Post("/pullRequest/reassign", prHandler.ReassignReviewer)
	})

	// Admin-only endpoints (require JWT + admin team membership)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(authService))
		r.Use(middleware.AdminMiddleware())

		r.Post("/users/setIsActive", userHandler.SetIsActive)
		r.Post("/users/batchDeactivateTeam", userHandler.BatchDeactivateTeam)
		r.Post("/users/batchDeactivateUsers", userHandler.BatchDeactivateUsers)
		r.Get("/admin/users", userHandler.ListAllUsers)
		r.Get("/admin/teams", teamHandler.ListAllTeams)

		// Statistics endpoint
		r.Get("/statistics", statisticsHandler.GetStatistics)
	})

	return r
}
