package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
	"github.com/tainalo2/devis_generator/backend/internal/config"
	"github.com/tainalo2/devis_generator/backend/internal/crypto"
	"github.com/tainalo2/devis_generator/backend/internal/db"
	"github.com/tainalo2/devis_generator/backend/internal/httpapi"
)

// Version is injected at build time via -ldflags.
var Version = "dev"

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load(filepath.Join("..", ".env"))
	_ = godotenv.Load(filepath.Join("..", "..", ".env"))

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}

	box, err := crypto.NewBox(cfg.EncryptionKey)
	if err != nil {
		slog.Error("encryption", "error", err)
		os.Exit(1)
	}

	startupCtx, startupCancel := context.WithTimeout(context.Background(), cfg.StartupTimeout)
	defer startupCancel()

	retryOpts := db.DefaultRetryOptions()

	if err := db.RunMigrationsWithRetry(startupCtx, cfg.DatabaseURL, retryOpts); err != nil {
		slog.Error("migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	pool, err := db.NewPoolWithRetry(startupCtx, cfg.DatabaseURL, retryOpts)
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	staticDir := cfg.StaticDir
	if !filepath.IsAbs(staticDir) {
		if abs, err := filepath.Abs(staticDir); err == nil {
			staticDir = abs
		}
	}

	api := httpapi.New(httpapi.Options{
		DB:           pool,
		Store:        authstore.New(pool),
		Box:          box,
		StaticDir:    staticDir,
		Version:      Version,
		CookieSecure: cfg.CookieSecure,
		SessionTTL:   cfg.SessionTTL,
		Argon2:       cfg.Argon2,
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", cfg.Addr, "env", cfg.Environment, "version", Version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("stopped")
}
