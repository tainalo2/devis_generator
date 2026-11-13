package db

import (
	"strings"
	"testing"
)

func TestEmailSecretsDownMigrationRemovesIncompatibleRowsFirst(t *testing.T) {
	sqlBytes, err := embeddedMigrations.ReadFile("migrations/000014_email_secrets.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	deleteSecrets := strings.Index(sql, "DELETE FROM payment_secrets")
	deleteAudit := strings.Index(sql, "DELETE FROM payment_secret_audit")
	alterSecrets := strings.Index(sql, "ALTER TABLE payment_secrets")
	if deleteSecrets < 0 || deleteAudit < 0 || alterSecrets < 0 || deleteSecrets > alterSecrets || deleteAudit > alterSecrets {
		t.Fatalf("rollback must delete email credentials and audit rows before narrowing constraints:\n%s", sql)
	}
}

func TestReminderSafeguardsRollbackDeduplicatesBeforeRestoringLegacyKey(t *testing.T) {
	sqlBytes, err := embeddedMigrations.ReadFile("migrations/000017_reminder_delivery_safeguards.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	dedupe := strings.Index(sql, "DELETE FROM reminders")
	legacyKey := strings.Index(sql, "reminders_user_id_idempotency_key_key")
	if dedupe < 0 || legacyKey < 0 || dedupe > legacyKey {
		t.Fatalf("rollback must deduplicate reminders before restoring the legacy account-wide key:\n%s", sql)
	}
}

func TestReminderSafeguardsCountEveryDeliveryAttempt(t *testing.T) {
	sqlBytes, err := embeddedMigrations.ReadFile("migrations/000017_reminder_delivery_safeguards.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(sqlBytes)
	if !strings.Contains(sql, "CREATE TABLE reminder_delivery_attempts") || !strings.Contains(sql, "reminder_delivery_attempts_rate_idx") {
		t.Fatalf("forward migration must persist every outbound attempt for rate limiting:\n%s", sql)
	}
}

func TestToMigrateURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"postgres://u:p@localhost:5432/db?sslmode=disable", "pgx5://u:p@localhost:5432/db?sslmode=disable"},
		{"postgresql://u:p@localhost/db", "pgx5://u:p@localhost/db"},
		{"pgx5://already", "pgx5://already"},
	}
	for _, tc := range cases {
		if got := toMigrateURL(tc.in); got != tc.want {
			t.Fatalf("toMigrateURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMigrateURLUsesAdvisoryLockDefault(t *testing.T) {
	ok := "postgres://u:p@localhost:5432/db?sslmode=disable"
	if !MigrateURLUsesAdvisoryLockDefault(ok) {
		t.Fatalf("expected advisory lock default for %q", ok)
	}

	disabled := "postgres://u:p@localhost:5432/db?x-lock-strategy=off"
	if MigrateURLUsesAdvisoryLockDefault(disabled) {
		t.Fatalf("expected advisory lock disabled for %q", disabled)
	}
}
