package httpapi

import (
	"net/http"
)

func (s *Server) handleStripeConnectOnboard(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		return
	}
	link, err := s.stripeConnect.Start(r.Context(), u.ID, u.Email)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Onboarding impossible.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"url": link})
}

func (s *Server) handleStripeConnectRefresh(w http.ResponseWriter, r *http.Request) {
	u, ok := UserFromContext(r.Context())
	if !ok {
		return
	}
	acc, err := s.stripeConnect.Refresh(r.Context(), u.ID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "Compte Stripe introuvable.")
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

func (s *Server) handleStripeConnectCallback(w http.ResponseWriter, r *http.Request) {
	s.handleStripeConnectRefresh(w, r)
}
