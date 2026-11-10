package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/doclink"
)

func TestDocumentLinkRedirectsReplacedQuoteToCurrentVersion(t *testing.T) {
	secret := []byte("test-doclink-secret")
	oldID := uuid.New()
	currentID := uuid.New()
	oldCreatedAt := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	currentCreatedAt := oldCreatedAt.Add(24 * time.Hour)
	oldToken, err := doclink.Generate(secret, oldID.String(), doclink.KindQuote, 1, oldCreatedAt)
	if err != nil {
		t.Fatal(err)
	}

	s := New(Options{DocLinkSecret: secret})
	s.resolveDocumentLink = func(_ context.Context, id uuid.UUID) (documentLinkTarget, error) {
		if id != oldID {
			t.Fatalf("resolved id = %s, want %s", id, oldID)
		}
		return documentLinkTarget{
			ID: currentID, Kind: doclink.KindQuote, Version: 2,
			CreatedAt: currentCreatedAt, Replaced: true,
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/d/"+oldToken, nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusTemporaryRedirect, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.HasSuffix(location, "?replaced=1") {
		t.Fatalf("Location = %q, want automatic replacement marker", location)
	}
	parts := strings.Split(strings.TrimPrefix(location, "/d/"), "?")
	payload, err := doclink.Verify(secret, parts[0])
	if err != nil {
		t.Fatalf("verify redirected token: %v", err)
	}
	if payload.DocID != currentID.String() || payload.Version != 2 || payload.Kind != doclink.KindQuote {
		t.Fatalf("redirect payload = %+v", payload)
	}
}

func TestDocumentLinkRendersVisibleReplacementMessage(t *testing.T) {
	secret := []byte("test-doclink-secret")
	id := uuid.New()
	createdAt := time.Now().UTC()
	token, err := doclink.Generate(secret, id.String(), doclink.KindQuote, 2, createdAt)
	if err != nil {
		t.Fatal(err)
	}

	s := New(Options{DocLinkSecret: secret})
	s.resolveDocumentLink = func(_ context.Context, got uuid.UUID) (documentLinkTarget, error) {
		return documentLinkTarget{ID: got, Kind: doclink.KindQuote, Version: 2, CreatedAt: createdAt}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/d/"+token+"?replaced=1", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("Content-Type = %q, want HTML", contentType)
	}
	if body := rec.Body.String(); !strings.Contains(body, doclink.ReplacedMessage) || !strings.Contains(body, "version actuelle") {
		t.Fatalf("replacement page does not explain the redirect: %s", body)
	}
}
