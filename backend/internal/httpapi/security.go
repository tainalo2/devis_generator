package httpapi

import (
	"mime"
	"net/http"
	"net/url"
	"strings"
)

// maxBodyBytes plafonne le corps des requêtes API (un document de 500 lignes
// tient largement dans 1 Mio).
const maxBodyBytes = 1 << 20

// contentSecurityPolicy : tout est servi par l'application elle-même (polices
// incluses). Les attributs style sont nécessaires au rendu React (aperçu A4).
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; object-src 'none'; " +
	"base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// securityHeaders ajoute les en-têtes de durcissement à toutes les réponses.
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		if s.cookieSecure {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// apiRequestGuard protège les requêtes API qui modifient l'état :
//   - corps limité à maxBodyBytes ;
//   - Origin, s'il est présent, doit être celui du site (bloque les
//     formulaires d'autres sites, y compris sur la connexion) ;
//   - un corps doit être en application/json : un formulaire HTML ne peut pas
//     l'envoyer sans requête de pré-vérification CORS.
func (s *Server) apiRequestGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && !sameOrigin(origin, r.Host) {
			writeAPIError(w, http.StatusForbidden, "Origine de la requête refusée.")
			return
		}
		if r.ContentLength != 0 && !isJSON(r.Header.Get("Content-Type")) {
			writeAPIError(w, http.StatusUnsupportedMediaType, "Le corps de la requête doit être en JSON.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sameOrigin(origin, host string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, host)
}

func isJSON(contentType string) bool {
	mt, _, err := mime.ParseMediaType(contentType)
	return err == nil && mt == "application/json"
}
