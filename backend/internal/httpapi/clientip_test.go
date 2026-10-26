package httpapi

import (
	"net"
	"net/http/httptest"
	"testing"
)

func TestClientIPIgnoresSpoofedHeaders(t *testing.T) {
	s := &Server{}
	r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	r.RemoteAddr = "203.0.113.7:4321"
	r.Header.Set("X-Forwarded-For", "10.0.0.1")
	r.Header.Set("X-Real-IP", "10.0.0.2")
	if got := s.clientIP(r); got != "203.0.113.7" {
		t.Fatalf("sans proxy de confiance, l'IP doit venir de la connexion : %q", got)
	}
}

func TestClientIPBehindTrustedProxy(t *testing.T) {
	_, proxies, _ := net.ParseCIDR("172.18.0.0/16")
	s := &Server{trustedProxies: []*net.IPNet{proxies}}
	r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	r.RemoteAddr = "172.18.0.5:4321"
	// Le client a injecté « 1.2.3.4 » ; le proxy ajoute la vraie adresse à droite.
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 198.51.100.9")
	if got := s.clientIP(r); got != "198.51.100.9" {
		t.Fatalf("derrière un proxy de confiance, attendu l'adresse ajoutée par le proxy : %q", got)
	}
}
