package migrate

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/cmd/jobs/internal"
)

func RootCommand() *cobra.Command {
	var migrationMode string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			migrationMode := config.AutoMigrate(migrationMode)
			if !migrationMode.Enabled() {
				return fmt.Errorf("migration mode is disabled")
			}

			if migrationMode == config.AutoMigrateMigrationJob {
				return fmt.Errorf("migration mode cannot be job for this command")
			}

			internal.App.Migrator.Config.AutoMigrate = migrationMode
			err := internal.App.Migrator.Migrate(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to migrate database: %w", err)
			}

			internal.App.Logger.Info("database migrated successfully")

			return nil
		},
	}

	cmd.Flags().StringVarP(&migrationMode, "mode", "m", string(config.AutoMigrateMigration), "Migration mode allowed values: ent, migration")

	return cmd
}
