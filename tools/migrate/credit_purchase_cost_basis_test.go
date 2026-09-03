package migrate_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func TestCreditPurchaseCostBasisMigrationRejectsNaN(t *testing.T) {
	t.Parallel()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer testDB.PGDriver.Close()

	tx, err := testDB.PGDriver.DB().BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, tx.Rollback())
	}()

	_, err = tx.Exec(`CREATE TEMP TABLE charge_credit_purchases ()`)
	require.NoError(t, err)

	_, err = tx.Exec(readMigration(t, "20260806163516_credit_purchase_cost_basis_bridge.up.sql"))
	require.NoError(t, err)

	_, err = tx.Exec(`
		INSERT INTO charge_credit_purchases (schema_level, fiat_cost_basis, settlement_type)
		VALUES (2, 'NaN'::numeric, 'invoice')
	`)
	require.ErrorContains(t, err, "fiat_cost_basis_positive")
}
