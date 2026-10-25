package db

import (
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
// databaseURL must use a scheme accepted by golang-migrate's pgx/v5 driver
// (postgres:// or pgx5://). postgres:// is rewritten to pgx5://.
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

func toMigrateURL(databaseURL string) string {
	for _, from := range []string{"postgres://", "postgresql://"} {
		if len(databaseURL) >= len(from) && databaseURL[:len(from)] == from {
			return "pgx5://" + databaseURL[len(from):]
		}
	}
	return databaseURL
}

