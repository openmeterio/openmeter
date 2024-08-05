package startup

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/config"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func DB(cfg config.PostgresConfig, db *db.Client) error {
	if cfg.AutoMigrate.Enabled() {
		switch cfg.AutoMigrate {
		case config.AutoMigrateEnt:
			if err := db.Schema.Create(context.Background()); err != nil {
				return fmt.Errorf("failed to migrate db: %w", err)
			}
		case config.AutoMigrateMigration:
			if err := migrate.Up(cfg.URL); err != nil {
				return fmt.Errorf("failed to migrate db: %w", err)
			}
		}
	}
	return nil
}
