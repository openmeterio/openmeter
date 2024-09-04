package startup

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func DB(ctx context.Context, cfg config.PostgresConfig, client *db.Client, db *sql.DB) error {
	if !cfg.AutoMigrate.Enabled() {
		return nil
	}

	switch cfg.AutoMigrate {
	case config.AutoMigrateEnt:
		if err := client.Schema.Create(ctx); err != nil {
			return fmt.Errorf("failed to migrate db: %w", err)
		}
	case config.AutoMigrateMigration:
		if m, err := migrate.Default(db); err == nil {
			if err := m.Up(); err != nil {
				return fmt.Errorf("failed to migrate db: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create migrate instance: %w", err)
		}
	}

	return nil
}
