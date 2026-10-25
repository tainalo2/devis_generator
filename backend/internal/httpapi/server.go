package httpapi

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server is the HTTP API and static file server.
type Server struct {
	db        Pinger
	staticDir string
	version   string
	router    chi.Router
}

// Options configures the HTTP server.
type Options struct {
	DB        Pinger
	StaticDir string
	Version   string
}

// New builds the chi router with health endpoints and SPA static serving.
func New(opts Options) *Server {
	s := &Server{
		db:        opts.DB,
		staticDir: opts.StaticDir,
		version:   opts.Version,
	}
	if s.version == "" {
		s.version = "dev"
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", s.handleHealthz)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthz", s.handleHealthz)
	})

	s.mountStatic(r)
	s.router = r
	return s
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) mountStatic(r chi.Router) {
	if s.staticDir == "" {
		return
	}

	info, err := os.Stat(s.staticDir)
	if err != nil || !info.IsDir() {
		r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{
				"service": "devisgenerator-vibed",
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

		// SPA fallback: serve index.html for client-side routes.
		index := filepath.Join(s.staticDir, "index.html")
		if _, err := os.Stat(index); err == nil {
			http.ServeFile(w, req, index)
			return
		}
		http.NotFound(w, req)
	})
}
