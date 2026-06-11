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
			if err := migratePostgres(cmd, migrationMode); err != nil {
				return err
			}

			if err := migrateClickHouse(cmd); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&migrationMode, "mode", "m", string(config.AutoMigrateMigration), "Migration mode allowed values: ent, migration")

	cmd.AddCommand(postgresCommand())
	cmd.AddCommand(clickhouseCommand())

	return cmd
}

func postgresCommand() *cobra.Command {
	var migrationMode string

	cmd := &cobra.Command{
		Use:   "postgres",
		Short: "Run PostgreSQL migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return migratePostgres(cmd, migrationMode)
		},
	}

	cmd.Flags().StringVarP(&migrationMode, "mode", "m", string(config.AutoMigrateMigration), "Migration mode allowed values: ent, migration")

	return cmd
}

func clickhouseCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clickhouse",
		Short: "Run ClickHouse migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return migrateClickHouse(cmd)
		},
	}
}

func migratePostgres(cmd *cobra.Command, migrationMode string) error {
	mode := config.AutoMigrate(migrationMode)
	if !mode.Enabled() {
		return fmt.Errorf("migration mode is disabled")
	}

	if mode == config.AutoMigrateMigrationJob {
		return fmt.Errorf("migration mode cannot be job for this command")
	}

	internal.App.Migrator.Config.AutoMigrate = mode
	if err := internal.App.Migrator.Migrate(cmd.Context()); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	internal.App.Logger.Info("database migrated successfully")

	return nil
}

func migrateClickHouse(cmd *cobra.Command) error {
	internal.App.ClickHouseMigrator.Config.AutoMigrate = config.ClickHouseAutoMigrateMigration
	if err := internal.App.ClickHouseMigrator.Migrate(cmd.Context()); err != nil {
		return fmt.Errorf("failed to migrate clickhouse: %w", err)
	}

	internal.App.Logger.Info("clickhouse migrated successfully")

	return nil
}
