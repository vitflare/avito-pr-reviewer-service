package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	config2 "pr-reviewer-service/pkg/config"

	_ "pr-reviewer-service/docs"
	"pr-reviewer-service/internal/handler"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/router"
	"pr-reviewer-service/internal/service"

	"github.com/go-playground/validator/v10"
)

// @title PR Reviewer Assignment Service API
// @version 1.0
// @description Service for automatic PR reviewer assignment
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	// Configure logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config2.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Connect to database
	pool, err := config2.MustInitDB(context.Background(), *cfg)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("successfully connected to database")

	// Initialize repositories
	authRepo := repository.NewAuthRepository(pool)
	teamRepo := repository.NewTeamRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	prRepo := repository.NewPRRepository(pool)
	statsRepo := repository.NewStatisticsRepository(pool)

	// Initialize validator
	validate := validator.New()

	// Initialize services
	authService := service.NewAuthService(authRepo, userRepo, cfg.JWTSecret)
	teamService := service.NewTeamService(teamRepo, userRepo)
	userService := service.NewUserService(userRepo, userRepo, prRepo)
	prService := service.NewPRService(prRepo, userRepo)
	statsService := service.NewStatisticsService(statsRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, validate)
	teamHandler := handler.NewTeamHandler(teamService, validate)
	userHandler := handler.NewUserHandler(userService, prService, validate)
	prHandler := handler.NewPRHandler(prService, validate)
	healthHandler := handler.NewHealthHandler()
	statisticsHandler := handler.NewStatisticsHandler(statsService)

	slog.Info("successfully configured services and handlers")

	// Setup router
	r := router.SetupRouter(
		authHandler,
		teamHandler,
		userHandler,
		prHandler,
		healthHandler,
		statisticsHandler,
		authService,
	)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("starting server", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}
