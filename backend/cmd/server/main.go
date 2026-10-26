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
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/config"
	"github.com/tainalo2/devis_generator/backend/internal/crypto"
	"github.com/tainalo2/devis_generator/backend/internal/db"
	"github.com/tainalo2/devis_generator/backend/internal/httpapi"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
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

	httpClient := legal.NewHTTPClient(cfg.LegalHTTPTimeout)
	recherche := legal.NewRechercheClient(httpClient)
	if cfg.LegalRechercheURL != "" {
		recherche.BaseURL = cfg.LegalRechercheURL
	}
	vies := legal.NewVIESClient(httpClient)
	if cfg.LegalVIESURL != "" {
		vies.URL = cfg.LegalVIESURL
	}
	tedb := legal.NewTEDBProvider(httpClient)
	if cfg.LegalTEDBURL != "" {
		tedb.URL = cfg.LegalTEDBURL
	}
	bce := legal.NewBCEProvider(httpClient)
	if cfg.LegalBCEURL != "" {
		bce.URL = cfg.LegalBCEURL
	}

	legalStore := legal.NewPGStore(pool)
	legalSvc := &legal.Service{
		Store:     legalStore,
		Provider:  legal.DefaultProvider{},
		Recherche: recherche,
		VIES:      vies,
		TEDB:      tedb,
		BCE:       bce,
		Stubs:     legal.OfficialStubs(),
		Logger:    slog.Default(),
	}

	bizStore := biz.NewStore(pool, box)
	bizStore.SetMentionsResolver(func(ctx context.Context) legal.Mentions {
		return legalSvc.MentionsForIssue(ctx)
	})
	if n, err := bizStore.SealLegacySnapshots(startupCtx); err != nil {
		slog.Error("chiffrement des instantanés émetteur", "error", err)
	} else if n > 0 {
		slog.Info("instantanés émetteur chiffrés", "documents", n)
	}
	if n, err := bizStore.SealClientAddresses(startupCtx); err != nil {
		slog.Error("chiffrement des adresses clients", "error", err)
	} else if n > 0 {
		slog.Info("adresses clients chiffrées", "clients", n)
	}
	if n, err := bizStore.SnapshotClientsOfIssuedDocuments(startupCtx); err != nil {
		slog.Error("instantanés client des documents émis", "error", err)
	} else if n > 0 {
		slog.Info("clients figés sur les documents émis", "documents", n)
	}

	api := httpapi.New(httpapi.Options{
		DB:    pool,
		Store: authstore.New(pool),
		Biz:   bizStore,
		Box:   box,
		Legal: legalSvc,
		PublicConfig: &httpapi.PublicSiteConfig{
			DonateURL:           cfg.DonateURL,
			LegalPublisherName:  cfg.LegalPublisherName,
			LegalPublisherAddr:  cfg.LegalPublisherAddr,
			LegalPublisherEmail: cfg.LegalPublisherEmail,
			LegalPublisherSIRET: cfg.LegalPublisherSIRET,
			LegalHostName:       cfg.LegalHostName,
			LegalHostAddr:       cfg.LegalHostAddr,
			LegalHostURL:        cfg.LegalHostURL,
		},
		StaticDir:      staticDir,
		Version:        Version,
		CookieSecure:   cfg.CookieSecure,
		SessionTTL:     cfg.SessionTTL,
		TrustedProxies: cfg.TrustedProxies,
		Argon2:         cfg.Argon2,
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()
	if cfg.LegalRefreshEnabled {
		refresher := legal.NewRefresher(pool, legalSvc, cfg.LegalRefreshInterval)
		refresher.Logger = slog.Default()
		go refresher.Start(runCtx)
		slog.Info("legal refresh started", "interval", cfg.LegalRefreshInterval.String())
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
	runCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("stopped")
}
