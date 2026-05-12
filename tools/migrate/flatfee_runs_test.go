package migrate_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestFlatFeeRunsCreditOnlyMigration(t *testing.T) {
	namespace := "flatfee_runs_credit_only"
	customerID := ulid.Make().String()
	withCreditsChargeID := ulid.Make().String()
	withoutCreditsChargeID := ulid.Make().String()
	creditAllocationID := ulid.Make().String()
	ledgerTransactionGroupID := ulid.Make().String()

	servicePeriodFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	servicePeriodTo := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	runner{
		stops: stops{
			{
				version:   20260511120000,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					insertFlatFeeRunMigrationCustomer(t, db, namespace, customerID)
					insertCreditOnlyFlatFeeCharge(t, db, namespace, customerID, withCreditsChargeID, "with-credits", servicePeriodFrom, servicePeriodTo, "120")
					insertCreditOnlyFlatFeeCharge(t, db, namespace, customerID, withoutCreditsChargeID, "without-credits", servicePeriodFrom, servicePeriodTo, "80")

					_, err := db.Exec(`
						INSERT INTO "charge_flat_fee_credit_allocations" (
							"id",
							"amount",
							"service_period_from",
							"service_period_to",
							"ledger_transaction_group_id",
							"namespace",
							"created_at",
							"updated_at",
							"line_id",
							"charge_id",
							"sort_hint",
							"type",
							"corrects_realization_id"
						)
						VALUES ($1, 120, $2, $3, $4, $5, NOW(), NOW(), NULL, $6, 0, 'allocation', NULL)
					`, creditAllocationID, servicePeriodFrom, servicePeriodTo, ledgerTransactionGroupID, namespace, withCreditsChargeID)
					require.NoError(t, err)
				},
			},
			{
				version:   20260511201803,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					assertCreditOnlyFlatFeeRunBackfilled(t, db, namespace, withCreditsChargeID, "120", "120")
					assertCreditOnlyFlatFeeRunBackfilled(t, db, namespace, withoutCreditsChargeID, "80", "0")

					var allocationCount int
					err := db.QueryRow(`
						SELECT COUNT(*)
						FROM "charge_flat_fee_run_credit_allocations" AS "ca"
						JOIN "charge_flat_fee_runs" AS "r" ON "r"."id" = "ca"."run_id"
						WHERE "r"."namespace" = $1
						  AND "r"."charge_id" = $2
					`, namespace, withCreditsChargeID).Scan(&allocationCount)
					require.NoError(t, err)
					require.Equal(t, 1, allocationCount)

					var allocationAmount, allocationType, allocationLedgerTransactionGroupID, allocationRunID, currentRunID string
					err = db.QueryRow(`
						SELECT
							"ca"."amount"::text,
							"ca"."type",
							"ca"."ledger_transaction_group_id",
							"ca"."run_id",
							"ff"."current_realization_run_id"
						FROM "charge_flat_fee_run_credit_allocations" AS "ca"
						JOIN "charge_flat_fee_runs" AS "r" ON "r"."id" = "ca"."run_id"
						JOIN "charge_flat_fees" AS "ff" ON "ff"."id" = "r"."charge_id"
						WHERE "ca"."id" = $1
					`, creditAllocationID).Scan(&allocationAmount, &allocationType, &allocationLedgerTransactionGroupID, &allocationRunID, &currentRunID)
					require.NoError(t, err)
					require.Equal(t, "120", allocationAmount)
					require.Equal(t, "allocation", allocationType)
					require.Equal(t, ledgerTransactionGroupID, allocationLedgerTransactionGroupID)
					require.Equal(t, currentRunID, allocationRunID)

					err = db.QueryRow(`
						SELECT COUNT(*)
						FROM "charge_flat_fee_run_credit_allocations" AS "ca"
						JOIN "charge_flat_fee_runs" AS "r" ON "r"."id" = "ca"."run_id"
						WHERE "r"."namespace" = $1
						  AND "r"."charge_id" = $2
					`, namespace, withoutCreditsChargeID).Scan(&allocationCount)
					require.NoError(t, err)
					require.Zero(t, allocationCount)
				},
			},
		},
	}.Test(t)
}

func insertFlatFeeRunMigrationCustomer(t *testing.T, db *sql.DB, namespace, customerID string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO "customers" (
			"id",
			"key",
			"namespace",
			"created_at",
			"updated_at",
			"name"
		)
		VALUES ($1, 'flatfee-runs-customer', $2, NOW(), NOW(), 'Flat Fee Runs Customer')
	`, customerID, namespace)
	require.NoError(t, err)
}

func insertCreditOnlyFlatFeeCharge(t *testing.T, db *sql.DB, namespace, customerID, chargeID, uniqueReferenceID string, servicePeriodFrom, servicePeriodTo time.Time, amount string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO "charge_flat_fees" (
			"id",
			"namespace",
			"payment_term",
			"invoice_at",
			"settlement_mode",
			"pro_rating",
			"amount_before_proration",
			"amount_after_proration",
			"service_period_from",
			"service_period_to",
			"billing_period_from",
			"billing_period_to",
			"full_service_period_from",
			"full_service_period_to",
			"status",
			"unique_reference_id",
			"currency",
			"managed_by",
			"created_at",
			"updated_at",
			"name",
			"customer_id",
			"status_detailed"
		)
		VALUES (
			$1,
			$2,
			'in_advance',
			$4,
			'credit_only',
			'no_prorating',
			$7::numeric,
			$7::numeric,
			$3,
			$4,
			$3,
			$4,
			$3,
			$4,
			'final',
			$5,
			'USD',
			'subscription',
			NOW(),
			NOW(),
			$5,
			$6,
			'final'
		)
	`, chargeID, namespace, servicePeriodFrom, servicePeriodTo, uniqueReferenceID, customerID, amount)
	require.NoError(t, err)
}

func assertCreditOnlyFlatFeeRunBackfilled(t *testing.T, db *sql.DB, namespace, chargeID, amountAfterProration, creditedAmount string) {
	t.Helper()

	var runID, currentRunID, runType, initialType, runAmountAfterProration, amount, chargesTotal, creditsTotal, total string
	err := db.QueryRow(`
		SELECT
			"r"."id",
			"ff"."current_realization_run_id",
			"r"."type",
			"r"."initial_type",
			"r"."amount_after_proration"::text,
			"r"."amount"::text,
			"r"."charges_total"::text,
			"r"."credits_total"::text,
			"r"."total"::text
		FROM "charge_flat_fee_runs" AS "r"
		JOIN "charge_flat_fees" AS "ff" ON "ff"."id" = "r"."charge_id"
		WHERE "r"."namespace" = $1
		  AND "r"."charge_id" = $2
	`, namespace, chargeID).Scan(&runID, &currentRunID, &runType, &initialType, &runAmountAfterProration, &amount, &chargesTotal, &creditsTotal, &total)
	require.NoError(t, err)
	require.NotEmpty(t, runID)
	require.Equal(t, runID, currentRunID)
	require.Equal(t, "final_realization", runType)
	require.Equal(t, "final_realization", initialType)
	require.Equal(t, amountAfterProration, runAmountAfterProration)
	require.Equal(t, creditedAmount, amount)
	require.Equal(t, creditedAmount, chargesTotal)
	require.Equal(t, creditedAmount, creditsTotal)
	require.Equal(t, "0", total)
}
