package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/tools/migrate/legacyent"
)

type databaseMigrationState int

const (
	databaseMigrationStateEmpty databaseMigrationState = iota
	databaseMigrationStateLegacyEnt
	databaseMigrationStateVersioned
	databaseMigrationStateUnknown
)

// AdoptLegacyEnt brings an unversioned database created by Ent to the frozen migration baseline.
// It intentionally stops at the baseline so the normal migration command remains responsible for
// upgrading from that version to the target OpenMeter version.
func AdoptLegacyEnt(ctx context.Context, db *sql.DB, connectionString string, logger *slog.Logger) error {
	if db == nil {
		return errors.New("database is required")
	}
	if connectionString == "" {
		return errors.New("connection string is required")
	}
	if logger == nil {
		return errors.New("logger is required")
	}

	lockConn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire legacy Ent migration connection: %w", err)
	}
	defer lockConn.Close()

	if _, err := lockConn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext('openmeter.legacy-ent-adoption'))`); err != nil {
		return fmt.Errorf("acquire legacy Ent migration lock: %w", err)
	}
	defer func() {
		if _, err := lockConn.ExecContext(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock(hashtext('openmeter.legacy-ent-adoption'))`); err != nil {
			logger.Error("failed to release legacy Ent migration lock", "error", err)
		}
	}()

	state, err := inspectDatabaseMigrationState(ctx, db)
	if err != nil {
		return err
	}

	switch state {
	case databaseMigrationStateEmpty:
		return errors.New("cannot adopt an empty database; run the normal migration command instead")
	case databaseMigrationStateVersioned:
		return errors.New("database is already managed by versioned migrations; run the normal migration command instead")
	case databaseMigrationStateLegacyEnt:
		logger.Info("legacy Ent database detected; applying frozen schema", "commit", legacyent.BaselineCommit, "baseline_version", legacyent.BaselineVersion)
		if err := legacyent.MigrateToBaseline(ctx, db); err != nil {
			return err
		}
		if err := legacyent.Reconcile(ctx, db); err != nil {
			return fmt.Errorf("reconcile legacy Ent database: %w", err)
		}
	case databaseMigrationStateUnknown:
		return errors.New("database has no schema_om migration state and is neither empty nor a recognized Ent-managed OpenMeter database")
	default:
		return fmt.Errorf("unsupported database migration state: %d", state)
	}

	migrator, err := New(MigrateOptions{ConnectionString: connectionString, Migrations: OMMigrationsConfig, Logger: logger})
	if err != nil {
		return fmt.Errorf("create versioned migrator: %w", err)
	}
	defer migrator.CloseOrLogError()

	if err := migrator.Force(legacyent.BaselineVersion); err != nil {
		return fmt.Errorf("record legacy Ent baseline version %d: %w", legacyent.BaselineVersion, err)
	}

	logger.Info("legacy Ent database adopted", "baseline_version", legacyent.BaselineVersion)

	return nil
}

func inspectDatabaseMigrationState(ctx context.Context, db *sql.DB) (databaseMigrationState, error) {
	var hasMigrationTable bool
	if err := db.QueryRowContext(ctx, `SELECT to_regclass('schema_om') IS NOT NULL`).Scan(&hasMigrationTable); err != nil {
		return 0, fmt.Errorf("inspect migration state table: %w", err)
	}
	if hasMigrationTable {
		return databaseMigrationStateVersioned, nil
	}

	var hasLegacyFingerprint bool
	if err := db.QueryRowContext(ctx, `
		SELECT to_regclass('features') IS NOT NULL
			AND to_regclass('meters') IS NOT NULL
			AND to_regclass('entitlements') IS NOT NULL
			AND to_regclass('customers') IS NOT NULL
	`).Scan(&hasLegacyFingerprint); err != nil {
		return 0, fmt.Errorf("inspect legacy Ent database fingerprint: %w", err)
	}
	if hasLegacyFingerprint {
		return databaseMigrationStateLegacyEnt, nil
	}

	var tableCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_tables WHERE schemaname = ANY(current_schemas(false))`).Scan(&tableCount); err != nil {
		return 0, fmt.Errorf("inspect database tables: %w", err)
	}
	if tableCount == 0 {
		return databaseMigrationStateEmpty, nil
	}

	return databaseMigrationStateUnknown, nil
}
