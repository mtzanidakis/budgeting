package main

import (
	"embed"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/manolis/budgeting/internal/auth"
	"github.com/manolis/budgeting/internal/config"
	"github.com/manolis/budgeting/internal/database"
	"github.com/manolis/budgeting/internal/handlers"
	"github.com/manolis/budgeting/internal/middleware"
)

//go:embed all:frontend
var staticFiles embed.FS

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set up logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Connect to database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		logger.Error("Failed to migrate database", "error", err)
		os.Exit(1)
	}

	// Initialize session store
	sessionStore := auth.NewSessionStore()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, sessionStore)
	actionsHandler := handlers.NewActionsHandler(db)
	usersHandler := handlers.NewUsersHandler(db)
	configHandler := handlers.NewConfigHandler(cfg.Currency)

	// Set up router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logging(logger))

	// Public routes
	r.Post("/api/login", authHandler.Login)
	r.Get("/api/config", configHandler.GetConfig)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(sessionStore))

		r.Post("/api/logout", authHandler.Logout)
		r.Get("/api/actions", actionsHandler.List)
		r.Post("/api/actions", actionsHandler.Create)
		r.Get("/api/users", usersHandler.List)
	})

	// Serve static files (SPA fallback to index.html)
	staticFS, err := fs.Sub(staticFiles, "frontend")
	if err != nil {
		logger.Error("Failed to load static files", "error", err)
		os.Exit(1)
	}

	fileServer := http.FileServer(http.FS(staticFS))
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	})

	// Start server
	addr := ":" + cfg.Port
	logger.Info("Starting server", "port", cfg.Port)

	if err := http.ListenAndServe(addr, r); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
