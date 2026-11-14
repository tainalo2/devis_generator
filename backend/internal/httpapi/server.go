package httpapi

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/auth"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
	"github.com/tainalo2/devis_generator/backend/internal/bankcoordsvault"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/crypto"
	"github.com/tainalo2/devis_generator/backend/internal/gmailmail"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
	"github.com/tainalo2/devis_generator/backend/internal/microsoftmail"
	"github.com/tainalo2/devis_generator/backend/internal/paymentflags"
	"github.com/tainalo2/devis_generator/backend/internal/payments"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
	"github.com/tainalo2/devis_generator/backend/internal/reminders"
	"github.com/tainalo2/devis_generator/backend/internal/smtpconfig"
	"github.com/tainalo2/devis_generator/backend/internal/stripeconnect"
)

// Server is the HTTP API and static file server.
type Server struct {
	db                   Pinger
	store                *authstore.Store
	apiKeyLookup         func(context.Context, string) (authstore.APIKey, authstore.User, error)
	biz                  *biz.Store
	box                  *crypto.Box
	legal                *legal.Service
	gmailMail            gmailMailer
	microsoftMail        microsoftMailer
	smtp                 *smtpconfig.Service
	reminders            *reminders.Service
	publicConfig         *PublicSiteConfig
	staticDir            string
	version              string
	router               chi.Router
	cookieSecure         bool
	sessionTTL           time.Duration
	argon2               auth.Argon2Params
	authLimiter          *RateLimiter
	accountLimiter       *RateLimiter
	legalGuestLimiter    *RateLimiter
	guestLimiter         *RateLimiter
	previewLimiter       *RateLimiter
	webhookIPLimiter     *RateLimiter
	webhookGlobalLimiter *RateLimiter
	trustedProxies       []*net.IPNet
	bankCoordRepo        *bankcoordsvault.Repository
	paymentFlags         *paymentflags.Store
	payments             *payments.Service
	stripeConnect        *stripeconnect.Service
	docLinkSecret        []byte
}

// Options configures the HTTP server.
type microsoftMailer interface {
	AuthorizationURL(uuid.UUID) (string, error)
	HandleCallback(context.Context, paymentsecrets.Actor, uuid.UUID, string, string) error
	Send(context.Context, paymentsecrets.Actor, uuid.UUID, microsoftmail.Message) error
	Status(context.Context, paymentsecrets.Actor, uuid.UUID) (bool, error)
	Disconnect(context.Context, paymentsecrets.Actor, uuid.UUID) error
}

type gmailMailer interface {
	AuthorizationURL(uuid.UUID) (string, error)
	HandleCallback(context.Context, paymentsecrets.Actor, uuid.UUID, string, string) error
	Send(context.Context, paymentsecrets.Actor, uuid.UUID, gmailmail.Message) error
	Status(context.Context, paymentsecrets.Actor, uuid.UUID) (bool, error)
	Disconnect(context.Context, paymentsecrets.Actor, uuid.UUID) error
}

type Options struct {
	DB                Pinger
	Store             *authstore.Store
	Biz               *biz.Store
	Box               *crypto.Box
	Legal             *legal.Service
	GmailMail         gmailMailer
	MicrosoftMail     microsoftMailer
	SMTP              *smtpconfig.Service
	Reminders         *reminders.Service
	PublicConfig      *PublicSiteConfig
	StaticDir         string
	Version           string
	CookieSecure      bool
	SessionTTL        time.Duration
	BankCoordRepo     *bankcoordsvault.Repository
	PaymentFlagsStore *paymentflags.Store
	Payments          *payments.Service
	StripeConnect     *stripeconnect.Service
	DocLinkSecret     []byte
	Argon2            auth.Argon2Params
	// TrustedProxies : seuls ces réseaux peuvent fournir X-Forwarded-For.
	TrustedProxies []*net.IPNet
}

// New builds the chi router with health, auth, business API and SPA.
func New(opts Options) *Server {
	s := &Server{
		db:                   opts.DB,
		store:                opts.Store,
		biz:                  opts.Biz,
		box:                  opts.Box,
		legal:                opts.Legal,
		gmailMail:            opts.GmailMail,
		microsoftMail:        opts.MicrosoftMail,
		smtp:                 opts.SMTP,
		reminders:            opts.Reminders,
		publicConfig:         opts.PublicConfig,
		staticDir:            opts.StaticDir,
		version:              opts.Version,
		cookieSecure:         opts.CookieSecure,
		sessionTTL:           opts.SessionTTL,
		argon2:               opts.Argon2,
		authLimiter:          NewRateLimiter(20, time.Minute),
		accountLimiter:       NewRateLimiter(10, 15*time.Minute),
		legalGuestLimiter:    NewRateLimiter(30, time.Minute),
		guestLimiter:         NewRateLimiter(20, time.Minute),
		previewLimiter:       NewRateLimiter(120, time.Minute),
		webhookIPLimiter:     NewRateLimiter(60, time.Minute),
		webhookGlobalLimiter: NewRateLimiter(600, time.Minute),
		trustedProxies:       opts.TrustedProxies,
		bankCoordRepo:        opts.BankCoordRepo,
		paymentFlags:         opts.PaymentFlagsStore,
		payments:             opts.Payments,
		stripeConnect:        opts.StripeConnect,
		docLinkSecret:        opts.DocLinkSecret,
	}
	if s.store != nil {
		s.apiKeyLookup = func(ctx context.Context, hash string) (authstore.APIKey, authstore.User, error) {
			key, user, err := s.store.GetAPIKeyByHash(ctx, hash)
			if err == nil {
				s.store.TouchAPIKey(ctx, key.ID)
			}
			return key, user, err
		}
	}
	if s.version == "" {
		s.version = "dev"
	}
	if len(s.docLinkSecret) == 0 {
		slog.Error("SESSION_SECRET environment variable is required but not set", "component", "httpapi")
		os.Exit(1)
	}
	if s.sessionTTL <= 0 {
		s.sessionTTL = 30 * 24 * time.Hour
	}
	if s.argon2.KeyLen == 0 {
		s.argon2 = auth.DefaultArgon2Params()
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(s.securityHeaders)

	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.apiRequestGuard)
		r.Get("/healthz", s.handleHealthz)
		r.Get("/readyz", s.handleReadyz)
		r.Get("/public/config", s.handlePublicConfig)
		if s.payments != nil {
			r.Post("/payments/webhooks/{provider}", s.handlePaymentWebhook)
		}

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

			r.Get("/auth/api-keys", s.requireSession(s.handleListAPIKeys))
			r.Post("/auth/api-keys", s.requireSession(s.handleCreateAPIKey))
			r.Delete("/auth/api-keys/{id}", s.requireSession(s.handleRevokeAPIKey))

			r.Post("/billing/preview", s.handlePreviewTotals)
			r.Post("/guest/pdf", s.handleGuestPDF)

			r.Get("/legal/status", s.handleLegalStatus)
			r.Get("/legal/companies/search", s.handleCompanySearch)
			r.Post("/legal/vat/validate", s.handleVATValidate)

			r.Get("/issuer", s.requireAuth(s.handleGetIssuer))
			r.Put("/issuer", s.requireAuth(s.handlePutIssuer))

			r.Get("/email/smtp", s.requireAuth(s.handleGetSMTPConfig))
			r.Post("/email/smtp", s.requireAuth(s.handleSaveSMTPConfig))
			r.Delete("/email/smtp", s.requireAuth(s.handleDeleteSMTPConfig))

			r.Get("/clients", s.requireAuth(s.handleListClients))
			r.Post("/clients", s.requireAuth(s.handleCreateClient))
			r.Get("/clients/{id}", s.requireAuth(s.handleGetClient))
			r.Put("/clients/{id}", s.requireAuth(s.handleUpdateClient))
			r.Delete("/clients/{id}", s.requireAuth(s.handleDeleteClient))

			r.Get("/documents", s.requireAuth(s.handleListDocuments))
			r.Post("/documents", s.requireAuth(s.handleCreateDocument))
			r.Get("/documents/{id}", s.requireAuth(s.handleGetDocument))
			r.Get("/documents/{id}/link", s.requireAuth(s.handleCreateDocumentLink))
			r.Put("/documents/{id}", s.requireAuth(s.handleUpdateDocument))
			r.Delete("/documents/{id}", s.requireAuth(s.handleDeleteDocument))
			r.Post("/documents/{id}/issue", s.requireAuth(s.handleIssueDocument))
			r.Post("/documents/{id}/status", s.requireAuth(s.handleDocumentStatus))
			r.Post("/documents/{id}/convert-to-invoice", s.requireAuth(s.handleConvertQuote))
			r.Post("/documents/{id}/deposit-invoice", s.requireAuth(s.handleCreateDeposit))
			r.Post("/documents/{id}/credit-note", s.requireAuth(s.handleCreateCreditNote))
			r.Get("/documents/{id}/pdf", s.requireAuth(s.handleDocumentPDF))
			if s.payments != nil {
				r.Get("/payments/config/{provider}", s.requireAuth(s.handleGetPaymentConfig))
				r.Put("/payments/config/{provider}", s.requireAuth(s.handlePutPaymentConfig))
				r.Delete("/payments/config/{provider}", s.requireAuth(s.handleDeletePaymentConfig))
				r.Post("/documents/{id}/checkout", s.requireAuth(s.handlePaymentCheckout))
				if s.stripeConnect != nil {
					r.Post("/payments/stripe-connect/onboard", s.requireAuth(s.handleStripeConnectOnboard))
					r.Post("/payments/stripe-connect/refresh", s.requireAuth(s.handleStripeConnectRefresh))
					r.Get("/payments/stripe-connect/callback", s.requireAuth(s.handleStripeConnectCallback))
				}
			}
			r.Post("/documents/{id}/reminders", s.requireAuth(s.handleSendReminder))
			r.Get("/documents/{id}/reminders", s.requireAuth(s.handleListReminders))
			r.Get("/email/delivery-mode", s.requireAuth(s.handleGetReminderDeliveryMode))
			r.Put("/email/delivery-mode", s.requireAuth(s.handleSetReminderDeliveryMode))
			r.Get("/reminder-templates", s.requireAuth(s.handleListReminderTemplates))
			r.Post("/reminder-templates", s.requireAuth(s.handleCreateReminderTemplate))
			r.Put("/reminder-templates/{id}", s.requireAuth(s.handleUpdateReminderTemplate))
			r.Delete("/reminder-templates/{id}", s.requireAuth(s.handleDeleteReminderTemplate))

			r.Get("/email/microsoft/authorize", s.requireAuth(s.handleMicrosoftAuthorize))
			r.Get("/email/microsoft/callback", s.requireAuth(s.handleMicrosoftCallback))
			r.Get("/email/microsoft/status", s.requireAuth(s.handleMicrosoftStatus))
			r.Post("/email/microsoft/send", s.requireAuth(s.handleMicrosoftSend))
			r.Delete("/email/microsoft", s.requireAuth(s.handleMicrosoftDisconnect))

			r.Get("/email/gmail/authorize", s.requireAuth(s.handleGmailAuthorize))
			r.Get("/email/gmail/callback", s.requireAuth(s.handleGmailCallback))
			r.Get("/email/gmail/status", s.requireAuth(s.handleGmailStatus))
			r.Post("/email/gmail/send", s.requireAuth(s.handleGmailSend))
			r.Delete("/email/gmail", s.requireAuth(s.handleGmailDisconnect))
		})
	})

	r.Get("/d/{token}", s.handleDocumentLink)
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
