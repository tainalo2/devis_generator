package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func guardStatus(t *testing.T, method, contentType, origin, body string) int {
	t.Helper()
	s := &Server{}
	h := s.apiRequestGuard(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	r := httptest.NewRequest(method, "http://devis.example/api/v1/auth/login", strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code
}

func TestAPIRequestGuard(t *testing.T) {
	cases := []struct {
		name, method, ct, origin, body string
		want                           int
	}{
		{"JSON même origine", "POST", "application/json", "http://devis.example", `{}`, 204},
		{"JSON sans Origin (client non navigateur)", "POST", "application/json; charset=utf-8", "", `{}`, 204},
		{"formulaire text/plain d'un autre site", "POST", "text/plain", "https://attaquant.example", `{"email":"a@b.c"}`, 403},
		{"text/plain même origine", "POST", "text/plain", "http://devis.example", `{}`, 415},
		{"JSON d'un autre site", "POST", "application/json", "https://attaquant.example", `{}`, 403},
		{"DELETE sans corps", "DELETE", "", "http://devis.example", ``, 204},
		{"GET non concerné", "GET", "", "https://attaquant.example", ``, 204},
	}
	for _, tc := range cases {
		if got := guardStatus(t, tc.method, tc.ct, tc.origin, tc.body); got != tc.want {
			t.Errorf("%s : statut %d, attendu %d", tc.name, got, tc.want)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	s := &Server{cookieSecure: true}
	h := s.securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	for _, k := range []string{"Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy", "Strict-Transport-Security"} {
		if w.Header().Get(k) == "" {
			t.Errorf("en-tête %s absent", k)
		}
	}
}
