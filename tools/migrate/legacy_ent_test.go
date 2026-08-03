package migrate_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
	"github.com/openmeterio/openmeter/tools/migrate/legacyent"
)

func TestLegacyEntReconciliationIsRerunnable(t *testing.T) {
	// given:
	// - an unversioned database created from the frozen Ent schema
	// when:
	// - reconciliation is applied twice after an interrupted adoption
	// then:
	// - all non-Ent objects exist and the baseline state remains valid
	db := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer db.Close(t)

	sqlDB := db.PGDriver.DB()
	sqlDB.SetMaxOpenConns(1)
	_, err := sqlDB.ExecContext(t.Context(), `CREATE SCHEMA legacy_ent_reconciliation`)
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(t.Context(), `SET search_path TO legacy_ent_reconciliation`)
	require.NoError(t, err)

	require.NoError(t, legacyent.MigrateToBaseline(t.Context(), sqlDB))
	require.NoError(t, legacyent.Reconcile(t.Context(), sqlDB))
	require.NoError(t, legacyent.Reconcile(t.Context(), sqlDB))

	var viewExists bool
	require.NoError(t, sqlDB.QueryRowContext(t.Context(), `SELECT to_regclass('charges_search_v1s') IS NOT NULL`).Scan(&viewExists))
	require.True(t, viewExists)

	var writeSchemaLevel int
	require.NoError(t, sqlDB.QueryRowContext(t.Context(), `SELECT schema_level FROM billing_invoice_write_schema_levels WHERE id = 'write_schema_level'`).Scan(&writeSchemaLevel))
	require.Equal(t, 1, writeSchemaLevel)
}

func TestAdoptLegacyEnt(t *testing.T) {
	// given:
	// - an unversioned database created from the frozen Ent schema
	// when:
	// - the explicit adoption command runs
	// then:
	// - adoption records exactly the frozen baseline
	db := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer db.Close(t)

	require.NoError(t, legacyent.MigrateToBaseline(t.Context(), db.PGDriver.DB()))
	require.NoError(t, migrate.AdoptLegacyEnt(t.Context(), db.PGDriver.DB(), db.URL, testutils.NewLogger(t)))

	migrator, err := migrate.New(migrate.MigrateOptions{ConnectionString: db.URL, Migrations: migrate.OMMigrationsConfig, Logger: testutils.NewLogger(t)})
	require.NoError(t, err)
	defer migrator.CloseOrLogError()

	version, dirty, err := migrator.Version()
	require.NoError(t, err)
	require.False(t, dirty)
	require.Equal(t, legacyent.BaselineVersion, version)
}

func TestMigrateFromLegacyEntBaselineToLatest(t *testing.T) {
	// given:
	// - a database at the frozen Ent schema with reconciliation applied and the baseline recorded
	// when:
	// - the normal Atlas migration runs
	// then:
	// - the database reaches the latest embedded migration version
	db := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer db.Close(t)

	require.NoError(t, legacyent.MigrateToBaseline(t.Context(), db.PGDriver.DB()))
	require.NoError(t, legacyent.Reconcile(t.Context(), db.PGDriver.DB()))

	migrator, err := migrate.New(migrate.MigrateOptions{ConnectionString: db.URL, Migrations: migrate.OMMigrationsConfig, Logger: testutils.NewLogger(t)})
	require.NoError(t, err)
	defer migrator.CloseOrLogError()
	require.NoError(t, migrator.Force(legacyent.BaselineVersion))

	latestVersion, err := migrator.LatestVersion()
	require.NoError(t, err)
	require.NoError(t, migrator.Up())

	version, dirty, err := migrator.Version()
	require.NoError(t, err)
	require.False(t, dirty)
	require.Equal(t, latestVersion, version)
}

func TestAdoptLegacyEntRejectsVersionedDatabase(t *testing.T) {
	// given:
	// - a database whose schema_om version predates the frozen Ent baseline
	// when:
	// - the explicit adoption command is invoked
	// then:
	// - it refuses to overwrite the existing migration history
	db := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer db.Close(t)

	migrator, err := migrate.New(migrate.MigrateOptions{ConnectionString: db.URL, Migrations: migrate.OMMigrationsConfig, Logger: testutils.NewLogger(t)})
	require.NoError(t, err)
	require.NoError(t, migrator.Migrate(20240826120919))
	migrator.CloseOrLogError()

	err = migrate.AdoptLegacyEnt(t.Context(), db.PGDriver.DB(), db.URL, testutils.NewLogger(t))
	require.ErrorContains(t, err, "already managed by versioned migrations")
}

func TestAdoptLegacyEntRejectsEmptyDatabase(t *testing.T) {
	// given:
	// - an empty database
	// when:
	// - the explicit adoption command is invoked
	// then:
	// - it directs the operator to normal migrations
	db := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer db.Close(t)

	err := migrate.AdoptLegacyEnt(t.Context(), db.PGDriver.DB(), db.URL, testutils.NewLogger(t))
	require.ErrorContains(t, err, "cannot adopt an empty database")
}

func TestAdoptLegacyEntRejectsUnknownUnversionedDatabase(t *testing.T) {
	// given:
	// - a non-empty unversioned database that is not recognizable as OpenMeter
	// when:
	// - the migration job inspects it
	// then:
	// - it refuses to claim the OpenMeter baseline
	db := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer db.Close(t)

	_, err := db.PGDriver.DB().ExecContext(t.Context(), `CREATE TABLE operator_owned_table (id bigint PRIMARY KEY)`)
	require.NoError(t, err)

	err = migrate.AdoptLegacyEnt(t.Context(), db.PGDriver.DB(), db.URL, testutils.NewLogger(t))
	require.ErrorContains(t, err, "neither empty nor a recognized Ent-managed OpenMeter database")
}
