package httpapi

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tainalo2/devis_generator/backend/internal/auth"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
	"github.com/tainalo2/devis_generator/backend/internal/crypto"
)

// Server is the HTTP API and static file server.
type Server struct {
	db           Pinger
	store        *authstore.Store
	box          *crypto.Box
	staticDir    string
	version      string
	router       chi.Router
	cookieSecure bool
	sessionTTL   time.Duration
	argon2       auth.Argon2Params
	authLimiter  *RateLimiter
}

// Options configures the HTTP server.
type Options struct {
	DB           Pinger
	Store        *authstore.Store
	Box          *crypto.Box
	StaticDir    string
	Version      string
	CookieSecure bool
	SessionTTL   time.Duration
	Argon2       auth.Argon2Params
}

// New builds the chi router with health, auth and SPA static serving.
func New(opts Options) *Server {
	s := &Server{
		db:           opts.DB,
		store:        opts.Store,
		box:          opts.Box,
		staticDir:    opts.StaticDir,
		version:      opts.Version,
		cookieSecure: opts.CookieSecure,
		sessionTTL:   opts.SessionTTL,
		argon2:       opts.Argon2,
		authLimiter:  NewRateLimiter(20, time.Minute),
	}
	if s.version == "" {
		s.version = "dev"
	}
	if s.sessionTTL <= 0 {
		s.sessionTTL = 30 * 24 * time.Hour
	}
	if s.argon2.KeyLen == 0 {
		s.argon2 = auth.DefaultArgon2Params()
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", s.handleHealthz)
		r.Get("/readyz", s.handleReadyz)

		r.Group(func(r chi.Router) {
			r.Use(s.withOptionalSession)
			r.Use(s.requireCSRF)

			r.Post("/auth/register", s.handleRegister)
			r.Post("/auth/login", s.handleLogin)
			r.Post("/auth/logout", s.handleLogout)
			r.Post("/auth/password-reset/request", s.handlePasswordResetRequest)

			r.Get("/auth/me", s.requireAuth(s.handleMe))
			r.Get("/auth/export", s.requireAuth(s.handleExport))
			r.Delete("/auth/me", s.requireAuth(s.handleDeleteAccount))
		})
	})

	s.mountStatic(r)
	s.router = r
	return s
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.router
}

// Box returns the application encryption box (may be nil in tests).
func (s *Server) Box() *crypto.Box { return s.box }

func (s *Server) mountStatic(r chi.Router) {
	if s.staticDir == "" {
		return
	}

	info, err := os.Stat(s.staticDir)
	if err != nil || !info.IsDir() {
		r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{
				"service": "Devis Generator : VIBED",
				"hint":    "frontend build not found; run npm run build in frontend/",
			})
		})
		return
	}

	fileServer := http.FileServer(http.Dir(s.staticDir))
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		urlPath := path.Clean(req.URL.Path)
		if urlPath == "/" {
			fileServer.ServeHTTP(w, req)
			return
		}

		full := filepath.Join(s.staticDir, filepath.FromSlash(strings.TrimPrefix(urlPath, "/")))
		if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
			fileServer.ServeHTTP(w, req)
			return
		}

		index := filepath.Join(s.staticDir, "index.html")
		if _, err := os.Stat(index); err == nil {
			http.ServeFile(w, req, index)
			return
		}
		http.NotFound(w, req)
	})
}
