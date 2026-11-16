package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"net/mail"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

// collaboratorDTO represents a collaborator on an account (API response)
type collaboratorDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// invitationDTO represents a pending invitation (API response)
type invitationDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

// handleListCollaborators lists all collaborators for the authenticated user's account
func (s *Server) handleListCollaborators(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}

	// Query collaborators from the account
	rows, err := s.biz.ListCollaborators(r.Context(), user.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}

	var collaborators []collaboratorDTO
	for _, c := range rows {
		collaborators = append(collaborators, collaboratorDTO{
			ID:        c.ID.String(),
			Email:     c.Email,
			Role:      c.Role,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"collaborators": collaborators})
}

// handleListPendingInvitations lists all pending invitations for the account
func (s *Server) handleListPendingInvitations(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}

	rows, err := s.biz.ListPendingInvitations(r.Context(), user.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}

	var invitations []invitationDTO
	for _, inv := range rows {
		invitations = append(invitations, invitationDTO{
			ID:        inv.ID.String(),
			Email:     inv.Email,
			Role:      inv.Role,
			ExpiresAt: inv.ExpiresAt.Format(time.RFC3339),
			CreatedAt: inv.CreatedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"invitations": invitations})
}

// handleInviteCollaborator creates a new invitation for a collaborator and sends an email
func (s *Server) handleInviteCollaborator(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}

	var input struct {
		Email string `json:"email"`
	}

	if err := decodeJSON(r, &input); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Requête invalide.")
		return
	}

	// Validate email format
	if _, err := mail.ParseAddress(input.Email); err != nil {
		writeAPIError(w, http.StatusBadRequest, "Adresse email invalide.")
		return
	}

	// Permission check: only owner can invite
	isOwner, err := s.biz.IsUserOwner(r.Context(), user.ID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	if !isOwner {
		writeAPIError(w, http.StatusForbidden, "Seul le propriétaire du compte peut inviter des collaborateurs.")
		return
	}

	// Generate random token (32 bytes hex)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Create invitation with 7-day expiration
	invitationID := uuid.New()
	expiresAt := time.Now().AddDate(0, 0, 7)

	inv, err := s.biz.CreateInvitation(r.Context(), invitationID, user.ID, input.Email, "collaborator", token, expiresAt, user.ID)
	if err != nil {
		if errors.Is(err, biz.ErrDuplicateInvitation) {
			writeAPIError(w, http.StatusConflict, "Une invitation active existe déjà pour cet email.")
			return
		}
		if errors.Is(err, biz.ErrCannotInviteSelf) {
			writeAPIError(w, http.StatusBadRequest, "Vous ne pouvez pas vous inviter vous-même.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}

	// Send invitation email (best effort - don't fail if email sending fails)
	if s.smtp != nil {
		actor := paymentsecrets.Actor{UserID: user.ID, RequestID: middleware.GetReqID(r.Context())}
		
		// Build invite link (frontend should handle the /accept-invite/{token} route)
		inviteLink := "/accept-invite/" + token
		subject := "Invitation à collaborer"
		htmlBody := buildInvitationEmailHTML(user.Email, inviteLink, user.Email)

		// Try to send email, but don't fail the request if it fails
		_ = s.smtp.SendMail(r.Context(), actor, user.ID, input.Email, subject, htmlBody)
	}

	// Log the invitation action
	actor := paymentsecrets.Actor{UserID: user.ID, RequestID: middleware.GetReqID(r.Context())}
	_ = s.biz.LogAction(r.Context(), actor, "invite_collaborator", input.Email)

	// Return the invitation details
	writeJSON(w, http.StatusCreated, map[string]any{
		"invite_id":  inv.ID.String(),
		"email":      inv.Email,
		"expires_at": inv.ExpiresAt.Format(time.RFC3339),
	})
}

// buildInvitationEmailHTML constructs the HTML body for invitation emails
func buildInvitationEmailHTML(senderName, inviteLink, senderEmail string) string {
	return `<html>
<body style="font-family: Arial, sans-serif; color: #333;">
<h2>Vous êtes invité à collaborer</h2>
<p>Bonjour,</p>
<p><strong>` + senderName + `</strong> (` + senderEmail + `) vous invite à collaborer sur un compte.</p>
<p>
	<a href="` + inviteLink + `" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; display: inline-block;">
		Accepter l'invitation
	</a>
</p>
<p>Ou copiez ce lien: ` + inviteLink + `</p>
<p>Cette invitation expire dans 7 jours.</p>
<hr />
<p style="font-size: 12px; color: #666;">Devis Generator</p>
</body>
</html>`
}

// handleRevokeCollaborator removes a collaborator from an account
func (s *Server) handleRevokeCollaborator(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}

	collaboratorID := chi.URLParam(r, "id")
	if collaboratorID == "" {
		writeAPIError(w, http.StatusBadRequest, "ID du collaborateur manquant.")
		return
	}

	collabID, err := uuid.Parse(collaboratorID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "ID du collaborateur invalide.")
		return
	}

	// Revoke collaborator
	err = s.biz.RevokeCollaborator(r.Context(), user.ID, collabID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeAPIError(w, http.StatusNotFound, "Collaborateur introuvable.")
			return
		}
		if errors.Is(err, biz.ErrCannotRevokeLastOwner) {
			writeAPIError(w, http.StatusBadRequest, "Impossible de révoquer le dernier propriétaire.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}

	// Log the revocation
	actor := paymentsecrets.Actor{UserID: user.ID, RequestID: middleware.GetReqID(r.Context())}
	_ = s.biz.LogAction(r.Context(), actor, "revoke_collaborator", collabID.String())

	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// handleCancelInvitation cancels a pending invitation
func (s *Server) handleCancelInvitation(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "Non authentifié.")
		return
	}

	invitationID := chi.URLParam(r, "id")
	if invitationID == "" {
		writeAPIError(w, http.StatusBadRequest, "ID de l'invitation manquant.")
		return
	}

	invID, err := uuid.Parse(invitationID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "ID de l'invitation invalide.")
		return
	}

	// Cancel invitation
	err = s.biz.CancelInvitation(r.Context(), user.ID, invID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeAPIError(w, http.StatusNotFound, "Invitation introuvable.")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, "Erreur interne.")
		return
	}

	// Log the cancellation
	actor := paymentsecrets.Actor{UserID: user.ID, RequestID: middleware.GetReqID(r.Context())}
	_ = s.biz.LogAction(r.Context(), actor, "cancel_invitation", invID.String())

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
