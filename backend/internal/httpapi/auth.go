package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tainalo2/devis_generator/backend/internal/auth"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
)

const (
	sessionCookieName = "vibed_session"
	csrfCookieName    = "vibed_csrf"
	csrfHeaderName    = "X-CSRF-Token"
	minPasswordLen    = 8
	maxPasswordLen    = 128
	maxEmailLen       = 254
	maxNameLen        = 120
)

type contextKey string

const (
	ctxUserKey    contextKey = "user"
	ctxSessionKey contextKey = "session"
)

type authCredentials struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type userDTO struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

func userToDTO(u authstore.User) userDTO {
	return userDTO{
		ID:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		CreatedAt:   u.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.authLimiter.Allow("register:" + s.clientIP(r)) {
		writeAPIError(w, http.StatusTooManyRequests, "Trop de tentatives. Réessayez plus tard.")
		return
	}
	var body authCredentials
	if err := decodeJSON(r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	email, err := normalizeEmail(body.Email)
	if err != nil || !validPassword(body.Password) {
		writeAPIError(w, http.StatusBadRequest, "Impossible de créer le compte avec ces informations.")
		return
	}
	name := strings.TrimSpace(body.DisplayName)
	if utf8.RuneCountInString(name) > maxNameLen {
		writeAPIError(w, http.StatusBadRequest, "Impossible de créer le compte avec ces informations.")
		return
	}
	if s.store == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Service indisponible.")
		return
	}

	hash, err := auth.HashPassword(body.Password, s.argon2)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	u, err := s.store.CreateUser(r.Context(), email, hash, name)
	if err != nil {
		if errors.Is(err, authstore.ErrEmailTaken) {
			// Same message as other failures — no email enumeration.
			writeAPIError(w, http.StatusBadRequest, "Impossible de créer le compte avec ces informations.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	if err := s.issueSession(w, r, u); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": userToDTO(u)})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.authLimiter.Allow("login:" + s.clientIP(r)) {
		writeAPIError(w, http.StatusTooManyRequests, "Trop de tentatives. Réessayez plus tard.")
		return
	}
	var body authCredentials
	if err := decodeJSON(r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}
	email, err := normalizeEmail(body.Email)
	if err != nil || body.Password == "" {
		writeAPIError(w, http.StatusUnauthorized, "Identifiants incorrects.")
		return
	}
	// Limite par compte, indépendante de l'IP : freine la force brute distribuée.
	if !s.accountLimiter.Allow("login:" + email) {
		writeAPIError(w, http.StatusTooManyRequests, "Trop de tentatives. Réessayez plus tard.")
		return
	}
	if s.store == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Service indisponible.")
		return
	}

	// Constant-ish path: always hash something if user missing.
	u, err := s.store.GetUserByEmail(r.Context(), email)
	ok := false
	if err == nil && u.PasswordHash != "" {
		ok, _ = auth.VerifyPassword(body.Password, u.PasswordHash)
	} else {
		_, _ = auth.HashPassword("dummy-password-for-timing", s.argon2)
	}
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Identifiants incorrects.")
		return
	}
	if err := s.issueSession(w, r, u); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": userToDTO(u)})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		_ = s.store.RevokeSession(r.Context(), auth.HashToken(c.Value))
	}
	clearAuthCookies(w, s.cookieSecure)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": userToDTO(u)})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	data, err := s.store.ExportUserData(r.Context(), u.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	if s.biz != nil {
		business, err := s.biz.ExportBusinessData(r.Context(), u.ID)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
			return
		}
		for k, v := range business {
			data[k] = v
		}
	}
	w.Header().Set("Content-Disposition", `attachment; filename="devisgenerator-vibed-export.json"`)
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}
	var body struct {
		Confirm string `json:"confirm"`
	}
	if err := decodeJSON(r, &body); err != nil || strings.ToUpper(strings.TrimSpace(body.Confirm)) != "SUPPRIMER" {
		writeAPIError(w, http.StatusBadRequest, `Pour confirmer, envoyez {"confirm":"SUPPRIMER"}.`)
		return
	}
	if err := s.store.DeleteUserHard(r.Context(), u.ID); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	clearAuthCookies(w, s.cookieSecure)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handlePasswordResetRequest is a stub: always returns the same message (no email sent).
func (s *Server) handlePasswordResetRequest(w http.ResponseWriter, r *http.Request) {
	if !s.authLimiter.Allow("reset:" + s.clientIP(r)) {
		writeAPIError(w, http.StatusTooManyRequests, "Trop de tentatives. Réessayez plus tard.")
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	_ = decodeJSON(r, &body)
	// Intentionally ignore whether the email exists; no SMTP wired yet.
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Si un compte correspond, un e-mail de réinitialisation sera envoyé lorsque l'envoi d'e-mails sera configuré.",
	})
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, u authstore.User) error {
	raw, hash, err := auth.NewToken(32)
	if err != nil {
		return err
	}
	csrf, _, err := auth.NewToken(32)
	if err != nil {
		return err
	}
	ip := s.clientIP(r)
	var ipPtr *string
	if ip != "" {
		ipPtr = &ip
	}
	expires := time.Now().Add(s.sessionTTL)
	_, err = s.store.CreateSession(r.Context(), u.ID, hash, csrf, expires, r.UserAgent(), ipPtr)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   int(s.sessionTTL.Seconds()),
	})
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    csrf,
		Path:     "/",
		HttpOnly: false,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   int(s.sessionTTL.Seconds()),
	})
	return nil
}

func clearAuthCookies(w http.ResponseWriter, secure bool) {
	past := time.Unix(0, 0)
	for _, name := range []string{sessionCookieName, csrfCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: name == sessionCookieName,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			Expires:  past,
			MaxAge:   -1,
		})
	}
}

func (s *Server) loadSession(r *http.Request) (authstore.Session, authstore.User, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return authstore.Session{}, authstore.User{}, false
	}
	sess, u, err := s.store.GetSessionByTokenHash(r.Context(), auth.HashToken(c.Value))
	if err != nil {
		return authstore.Session{}, authstore.User{}, false
	}
	return sess, u, true
}

func (s *Server) withOptionalSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sess, u, ok := s.loadSession(r); ok {
			ctx := context.WithValue(r.Context(), ctxUserKey, u)
			ctx = context.WithValue(ctx, ctxSessionKey, sess)
			r = r.WithContext(ctx)
			_ = s.store.TouchSession(r.Context(), sess.ID)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFromContext(r.Context()); !ok {
			writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
			return
		}
		next(w, r)
	}
}

func (s *Server) requireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			next.ServeHTTP(w, r)
			return
		}
		// Only enforce when a session exists (or for auth mutations after login).
		sess, ok := SessionFromContext(r.Context())
		header := r.Header.Get(csrfHeaderName)
		cookie, _ := r.Cookie(csrfCookieName)
		cookieVal := ""
		if cookie != nil {
			cookieVal = cookie.Value
		}
		if ok {
			if header == "" || header != sess.CSRFToken || header != cookieVal {
				writeAPIError(w, http.StatusForbidden, "Jeton CSRF invalide.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// UserFromContext returns the authenticated user if present.
func UserFromContext(ctx context.Context) (authstore.User, bool) {
	u, ok := ctx.Value(ctxUserKey).(authstore.User)
	return u, ok
}

// SessionFromContext returns the session if present.
func SessionFromContext(ctx context.Context) (authstore.Session, bool) {
	s, ok := ctx.Value(ctxSessionKey).(authstore.Session)
	return s, ok
}

func normalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || len(email) > maxEmailLen {
		return "", errors.New("invalid email")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", errors.New("invalid email")
	}
	return email, nil
}

func validPassword(pw string) bool {
	n := utf8.RuneCountInString(pw)
	return n >= minPasswordLen && n <= maxPasswordLen
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
