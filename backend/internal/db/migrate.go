package db

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed all:migrations
var embeddedMigrations embed.FS

// RunMigrations applies embedded SQL migrations against DATABASE_URL.
//
// The golang-migrate pgx/v5 driver acquires a PostgreSQL advisory lock
// (pg_advisory_lock) by default before migrating, so concurrent app replicas
// starting together are safe: waiters block until the lock holder finishes.
// Do not append x-lock-strategy=off (or disable advisory locking) to DATABASE_URL.
func RunMigrations(databaseURL string) error {
	source, err := iofs.New(embeddedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, toMigrateURL(databaseURL))
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// RunMigrationsWithRetry retries RunMigrations until the database accepts connections.
// Concurrent replicas remain safe thanks to migrate's advisory lock (see RunMigrations).
func RunMigrationsWithRetry(ctx context.Context, databaseURL string, opts RetryOptions) error {
	return withRetry(ctx, "migrations", opts, func(context.Context) error {
		return RunMigrations(databaseURL)
	})
}

func toMigrateURL(databaseURL string) string {
	for _, from := range []string{"postgres://", "postgresql://"} {
		if len(databaseURL) >= len(from) && databaseURL[:len(from)] == from {
			return "pgx5://" + databaseURL[len(from):]
		}
	}
	return databaseURL
}

// MigrateURLUsesAdvisoryLockDefault reports whether the migrate URL leaves the
// pgx/v5 default advisory lock strategy enabled (no explicit disable).
func MigrateURLUsesAdvisoryLockDefault(databaseURL string) bool {
	u := toMigrateURL(databaseURL)
	// Explicit disables we refuse to treat as "safe default".
	for _, bad := range []string{
		"x-lock-strategy=off",
		"x-lock-strategy=false",
		"x-lock-strategy=none",
		"x-advisory-locking=false",
	} {
		if containsFold(u, bad) {
			return false
		}
	}
	return true
}

func containsFold(s, substr string) bool {
	return len(s) >= len(substr) && (indexFold(s, substr) >= 0)
}

func indexFold(s, substr string) int {
	n, m := len(s), len(substr)
	if m == 0 {
		return 0
	}
	for i := 0; i+m <= n; i++ {
		ok := true
		for j := 0; j < m; j++ {
			a, b := s[i+j], substr[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}
