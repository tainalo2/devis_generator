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
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/crypto"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

// Server is the HTTP API and static file server.
type Server struct {
	db                Pinger
	store             *authstore.Store
	biz               *biz.Store
	box               *crypto.Box
	legal             *legal.Service
	publicConfig      *PublicSiteConfig
	staticDir         string
	version           string
	router            chi.Router
	cookieSecure      bool
	sessionTTL        time.Duration
	argon2            auth.Argon2Params
	authLimiter       *RateLimiter
	legalGuestLimiter *RateLimiter
}

// Options configures the HTTP server.
type Options struct {
	DB           Pinger
	Store        *authstore.Store
	Biz          *biz.Store
	Box          *crypto.Box
	Legal        *legal.Service
	PublicConfig *PublicSiteConfig
	StaticDir    string
	Version      string
	CookieSecure bool
	SessionTTL   time.Duration
	Argon2       auth.Argon2Params
}

// New builds the chi router with health, auth, business API and SPA.
func New(opts Options) *Server {
	s := &Server{
		db:                opts.DB,
		store:             opts.Store,
		biz:               opts.Biz,
		box:               opts.Box,
		legal:             opts.Legal,
		publicConfig:      opts.PublicConfig,
		staticDir:         opts.StaticDir,
		version:           opts.Version,
		cookieSecure:      opts.CookieSecure,
		sessionTTL:        opts.SessionTTL,
		argon2:            opts.Argon2,
		authLimiter:       NewRateLimiter(20, time.Minute),
		legalGuestLimiter: NewRateLimiter(30, time.Minute),
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
		r.Get("/public/config", s.handlePublicConfig)

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

			r.Post("/billing/preview", s.handlePreviewTotals)
			r.Post("/guest/pdf", s.handleGuestPDF)

			r.Get("/legal/status", s.handleLegalStatus)
			r.Get("/legal/companies/search", s.handleCompanySearch)
			r.Post("/legal/vat/validate", s.handleVATValidate)

			r.Get("/issuer", s.requireAuth(s.handleGetIssuer))
			r.Put("/issuer", s.requireAuth(s.handlePutIssuer))

			r.Get("/clients", s.requireAuth(s.handleListClients))
			r.Post("/clients", s.requireAuth(s.handleCreateClient))
			r.Get("/clients/{id}", s.requireAuth(s.handleGetClient))
			r.Put("/clients/{id}", s.requireAuth(s.handleUpdateClient))
			r.Delete("/clients/{id}", s.requireAuth(s.handleDeleteClient))

			r.Get("/documents", s.requireAuth(s.handleListDocuments))
			r.Post("/documents", s.requireAuth(s.handleCreateDocument))
			r.Get("/documents/{id}", s.requireAuth(s.handleGetDocument))
			r.Put("/documents/{id}", s.requireAuth(s.handleUpdateDocument))
			r.Delete("/documents/{id}", s.requireAuth(s.handleDeleteDocument))
			r.Post("/documents/{id}/issue", s.requireAuth(s.handleIssueDocument))
			r.Post("/documents/{id}/convert-to-invoice", s.requireAuth(s.handleConvertQuote))
			r.Post("/documents/{id}/deposit-invoice", s.requireAuth(s.handleCreateDeposit))
			r.Post("/documents/{id}/credit-note", s.requireAuth(s.handleCreateCreditNote))
			r.Get("/documents/{id}/pdf", s.requireAuth(s.handleDocumentPDF))
		})
	})

	s.mountStatic(r)
	s.router = r
	return s
}

func (s *Server) Handler() http.Handler { return s.router }

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
