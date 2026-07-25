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
			if migrationMode == "ent" {
				return fmt.Errorf("ent migration is no longer supported; run 'openmeter-jobs migrate adopt-ent' once, then set migration mode to 'migration'")
			}
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

	cmd.Flags().StringVarP(&migrationMode, "mode", "m", string(config.AutoMigrateMigration), "Migration mode allowed values: migration")
	cmd.AddCommand(adoptEntCommand())

	return cmd
}

func adoptEntCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "adopt-ent",
		Short: "Adopt a database previously managed by Ent into versioned migrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := internal.App.Migrator.AdoptLegacyEnt(cmd.Context()); err != nil {
				return fmt.Errorf("failed to adopt legacy Ent database: %w", err)
			}

			internal.App.Logger.Info("legacy Ent database adopted successfully; run the normal migration command to upgrade to the target version")

			return nil
		},
	}
}
