package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/authstore"
)

// Without a reminders service (not configured), every reminders route must answer 503,
// never panic: a nil pointer used to turn into a bare 500 on a deployment without wiring.
func TestReminderRoutesAnswer503WhenServiceIsNotConfigured(t *testing.T) {
	s := &Server{}
	ctx := context.WithValue(context.Background(), ctxUserKey, authstore.User{ID: uuid.New()})
	id := uuid.New().String()
	cases := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		body    string
	}{
		{"mode get", s.handleGetReminderDeliveryMode, http.MethodGet, ""},
		{"mode set", s.handleSetReminderDeliveryMode, http.MethodPut, `{"mode":"smtp"}`},
		{"templates list", s.handleListReminderTemplates, http.MethodGet, ""},
		{"templates create", s.handleCreateReminderTemplate, http.MethodPost, `{"name":"n","subject":"s","html":"h"}`},
		{"templates update", s.handleUpdateReminderTemplate, http.MethodPut, `{"name":"n","subject":"s","html":"h"}`},
		{"templates delete", s.handleDeleteReminderTemplate, http.MethodDelete, ""},
		{"reminders list", s.handleListReminders, http.MethodGet, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(c.method, "/x/"+id, strings.NewReader(c.body)).WithContext(ctx)
			rec := httptest.NewRecorder()
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("panic: %v", r)
					}
				}()
				c.handler(rec, req)
			}()
			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("status=%d body=%s, want 503", rec.Code, rec.Body)
			}
		})
	}
}
