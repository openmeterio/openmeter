package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
)

const invoiceCreditPurchaseStatusMigration = "20260806065936_migrate_invoice_credit_purchase_statuses.up.sql"

func TestMigrateInvoiceCreditPurchaseStatuses(t *testing.T) {
	up := readMigration(t, invoiceCreditPurchaseStatusMigration)

	t.Run("maps durable realization states", func(t *testing.T) {
		withInvoiceCreditPurchaseStatusMigrationTables(t, func(tx *sql.Tx) {
			_, err := tx.Exec(`
				INSERT INTO charge_credit_purchases (id, settlement, status, status_detailed) VALUES
					('invoice-pending', '{"type":"invoice"}', 'active', 'active'),
					('invoice-authorized', '{"type":"invoice"}', 'active', 'active'),
					('invoice-settled', '{"type":"invoice"}', 'active', 'active'),
					('invoice-already-detailed', '{"type":"invoice"}', 'active', 'active.payment.pending'),
					('external-active', '{"type":"external"}', 'active', 'active')
			`)
			require.NoError(t, err)

			_, err = tx.Exec(`
				INSERT INTO charge_credit_purchase_credit_grants (charge_id, transaction_group_id) VALUES
					('invoice-pending', 'grant-pending'),
					('invoice-authorized', 'grant-authorized'),
					('invoice-settled', 'grant-settled'),
					('invoice-already-detailed', 'grant-already-detailed'),
					('external-active', 'grant-external')
			`)
			require.NoError(t, err)

			_, err = tx.Exec(`
				INSERT INTO charge_credit_purchase_invoiced_payments (charge_id, status) VALUES
					('invoice-authorized', 'authorized'),
					('invoice-settled', 'settled')
			`)
			require.NoError(t, err)

			_, err = tx.Exec(up)
			require.NoError(t, err)

			rows, err := tx.Query(`
				SELECT id, status, status_detailed
				FROM charge_credit_purchases
			`)
			require.NoError(t, err)
			defer rows.Close()

			got := map[string][2]string{}
			for rows.Next() {
				var id, status, statusDetailed string
				require.NoError(t, rows.Scan(&id, &status, &statusDetailed))
				got[id] = [2]string{status, statusDetailed}
			}
			require.NoError(t, rows.Err())

			require.Equal(t, map[string][2]string{
				"invoice-pending":          {"active", "active.payment.pending"},
				"invoice-authorized":       {"active", "active.payment.authorized"},
				"invoice-settled":          {"final", "final"},
				"invoice-already-detailed": {"active", "active.payment.pending"},
				"external-active":          {"active", "active"},
			}, got)
		})
	})

	t.Run("rejects an active invoice purchase without a credit grant", func(t *testing.T) {
		withInvoiceCreditPurchaseStatusMigrationTables(t, func(tx *sql.Tx) {
			_, err := tx.Exec(`
				INSERT INTO charge_credit_purchases (id, settlement, status, status_detailed)
				VALUES ('invoice-without-grant', '{"type":"invoice"}', 'active', 'active')
			`)
			require.NoError(t, err)

			_, err = tx.Exec(up)
			require.ErrorContains(t, err, "have no active credit grant")
		})
	})

	t.Run("rejects an unsupported invoiced payment status", func(t *testing.T) {
		withInvoiceCreditPurchaseStatusMigrationTables(t, func(tx *sql.Tx) {
			_, err := tx.Exec(`
				INSERT INTO charge_credit_purchases (id, settlement, status, status_detailed)
				VALUES ('invoice-unsupported-payment', '{"type":"invoice"}', 'active', 'active')
			`)
			require.NoError(t, err)

			_, err = tx.Exec(`
				INSERT INTO charge_credit_purchase_credit_grants (charge_id, transaction_group_id)
				VALUES ('invoice-unsupported-payment', 'grant-unsupported-payment')
			`)
			require.NoError(t, err)

			_, err = tx.Exec(`
				INSERT INTO charge_credit_purchase_invoiced_payments (charge_id, status)
				VALUES ('invoice-unsupported-payment', 'unsupported')
			`)
			require.NoError(t, err)

			_, err = tx.Exec(up)
			require.ErrorContains(t, err, "have an unsupported invoiced payment status")
		})
	})
}

func withInvoiceCreditPurchaseStatusMigrationTables(t *testing.T, action func(tx *sql.Tx)) {
	t.Helper()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer testDB.PGDriver.Close()

	tx, err := testDB.PGDriver.DB().BeginTx(t.Context(), nil)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, tx.Rollback())
	}()

	_, err = tx.Exec(`
		CREATE TEMP TABLE charge_credit_purchases (
			id text PRIMARY KEY,
			settlement jsonb NOT NULL,
			status text NOT NULL,
			status_detailed text NOT NULL
		);

		CREATE TEMP TABLE charge_credit_purchase_credit_grants (
			charge_id text NOT NULL,
			transaction_group_id text NOT NULL,
			deleted_at timestamptz NULL
		);

		CREATE TEMP TABLE charge_credit_purchase_invoiced_payments (
			charge_id text NOT NULL,
			status text NOT NULL,
			deleted_at timestamptz NULL
		);
	`)
	require.NoError(t, err)

	action(tx)
}
