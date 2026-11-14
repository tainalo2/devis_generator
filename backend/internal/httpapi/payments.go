package httpapi

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/paymentbyo"
	"github.com/tainalo2/devis_generator/backend/internal/payments"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

func (s *Server) paymentActor(r *http.Request) (paymentsecrets.Actor, bool) {
	u, ok := UserFromContext(r.Context())
	return paymentsecrets.Actor{UserID: u.ID, RequestID: middleware.GetReqID(r.Context())}, ok
}
func providerParam(r *http.Request) (paymentbyo.Provider, bool) {
	p := paymentbyo.Provider(chi.URLParam(r, "provider"))
	for _, v := range paymentbyo.AllProviders() {
		if p == v {
			return p, true
		}
	}
	return p, false
}
func (s *Server) handlePutPaymentConfig(w http.ResponseWriter, r *http.Request) {
	a, ok := s.paymentActor(r)
	if !ok {
		return
	}
	p, ok := providerParam(r)
	if !ok {
		writeAPIError(w, 400, "Fournisseur invalide.")
		return
	}
	var in payments.ConfigInput
	if decodeJSON(r, &in) != nil {
		writeAPIError(w, 400, "Requête invalide.")
		return
	}
	in.Provider = p
	v, err := s.payments.PutConfig(r.Context(), a, in)
	if err != nil {
		writeAPIError(w, 400, "Configuration invalide.")
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) handleGetPaymentConfig(w http.ResponseWriter, r *http.Request) {
	a, _ := s.paymentActor(r)
	p, ok := providerParam(r)
	if !ok {
		writeAPIError(w, 400, "Fournisseur invalide.")
		return
	}
	v, err := s.payments.GetConfig(r.Context(), a, p)
	if errors.Is(err, payments.ErrNotFound) {
		writeAPIError(w, 404, "Configuration introuvable.")
		return
	}
	if err != nil {
		writeAPIError(w, 500, "Erreur interne.")
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) handleDeletePaymentConfig(w http.ResponseWriter, r *http.Request) {
	a, _ := s.paymentActor(r)
	p, ok := providerParam(r)
	if !ok {
		writeAPIError(w, 400, "Fournisseur invalide.")
		return
	}
	if err := s.payments.DeleteConfig(r.Context(), a, p); err != nil && !errors.Is(err, payments.ErrNotFound) {
		writeAPIError(w, 500, "Erreur interne.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) handlePaymentCheckout(w http.ResponseWriter, r *http.Request) {
	a, _ := s.paymentActor(r)
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIError(w, 400, "Identifiant invalide.")
		return
	}
	var in payments.CheckoutInput
	if decodeJSON(r, &in) != nil {
		writeAPIError(w, 400, "Requête invalide.")
		return
	}
	v, err := s.payments.Checkout(r.Context(), a, id, in)
	if errors.Is(err, payments.ErrNotFound) {
		writeAPIError(w, 404, "Facture introuvable.")
		return
	}
	if err != nil {
		writeAPIError(w, 400, "Paiement impossible.")
		return
	}
	writeJSON(w, http.StatusCreated, v)
}
func (s *Server) handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	p, ok := providerParam(r)
	if !ok {
		writeAPIError(w, 404, "Fournisseur invalide.")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeAPIError(w, 400, "Requête invalide.")
		return
	}
	headers := map[string]string{
		"Stripe-Signature":         r.Header.Get("Stripe-Signature"),
		"Paypal-Transmission-Id":   r.Header.Get("Paypal-Transmission-Id"),
		"Paypal-Transmission-Time": r.Header.Get("Paypal-Transmission-Time"),
		"Paypal-Cert-Url":          r.Header.Get("Paypal-Cert-Url"),
		"Paypal-Auth-Algo":         r.Header.Get("Paypal-Auth-Algo"),
		"Paypal-Transmission-Sig":  r.Header.Get("Paypal-Transmission-Sig"),
	}
	if err = s.payments.ProcessWebhook(r.Context(), p, headers, body); err != nil {
		writeAPIError(w, 400, "Webhook invalide.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
