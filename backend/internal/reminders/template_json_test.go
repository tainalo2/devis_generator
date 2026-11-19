package reminders

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// The API returns Template as-is: its JSON keys are a contract with the frontend.
func TestTemplateJSONContract(t *testing.T) {
	raw, err := json.Marshal(Template{ID: uuid.New(), OwnerID: uuid.New(), Name: "Relance 1", Subject: "Sujet", HTML: "<p>x</p>"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "name", "subject", "html", "created_at", "updated_at"} {
		if _, ok := got[key]; !ok {
			t.Errorf("clé %q absente: %s", key, raw)
		}
	}
	for _, leaked := range []string{"OwnerID", "owner_id", "ID", "Subject", "HTML"} {
		if _, ok := got[leaked]; ok {
			t.Errorf("clé %q ne doit pas être exposée: %s", leaked, raw)
		}
	}
	var in Template
	if err := json.Unmarshal([]byte(`{"name":"N","subject":"S","html":"<b>h</b>"}`), &in); err != nil || in.Subject != "S" || in.HTML != "<b>h</b>" || in.Name != "N" {
		t.Fatalf("décodage: %v %+v", err, in)
	}
}
