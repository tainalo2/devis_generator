package httpapi

import "net/http"

// PublicSiteConfig is exposed to the SPA (no secrets).
type PublicSiteConfig struct {
	DonateURL           string `json:"donate_url"`
	LegalPublisherName  string `json:"legal_publisher_name"`
	LegalPublisherAddr  string `json:"legal_publisher_address"`
	LegalPublisherEmail string `json:"legal_publisher_email"`
	LegalPublisherSIRET string `json:"legal_publisher_siret"`
	LegalHostName       string `json:"legal_host_name"`
	LegalHostAddr       string `json:"legal_host_address"`
	LegalHostURL        string `json:"legal_host_url"`
}

func (s *Server) handlePublicConfig(w http.ResponseWriter, _ *http.Request) {
	cfg := s.publicConfig
	if cfg == nil {
		cfg = &PublicSiteConfig{}
	}
	writeJSON(w, http.StatusOK, cfg)
}
