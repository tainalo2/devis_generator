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
