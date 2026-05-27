package migrate_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestDedupeTaxCodesByAppMappingMigration(t *testing.T) {
	namespace := "dedupe_tax_codes_test"

	// Times used to control winner/loser ordering.
	// T1 < T2 so the older row wins when both are non-system.
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	// Group 1: same Stripe mapping txcd_10103001.
	//   Loser  (group1Loser):  key "stripe_txcd_10103001", non-system, created T1.
	//   Winner (group1Winner): key "saas_business",        system,     created T2.
	//   System wins despite being newer — exercises is_system DESC sort key.
	group1Loser := ulid.Make().String()
	group1Winner := ulid.Make().String()

	// Group 2: same Stripe mapping txcd_99999999, both non-system.
	//   Loser  (group2Loser):  key "stripe_txcd_99999999", created T2 (newer).
	//   Winner (group2Winner): key "legacy_99",             created T1 (older).
	//   No system row → oldest wins.
	group2Loser := ulid.Make().String()
	group2Winner := ulid.Make().String()

	// Singleton: a tax code with no duplicates — must remain live.
	singleton := ulid.Make().String()

	// Child-table rows pointing at group1Loser (FK rows that must be repointed).
	wfcRowID := ulid.Make().String() // billing_workflow_configs
	planID := ulid.Make().String()   // plans (parent for plan_rate_cards)
	phaseID := ulid.Make().String()  // plan_phases (parent for plan_rate_cards)
	prcRowID := ulid.Make().String() // plan_rate_cards

	customerID := ulid.Make().String() // customers (parent for subscriptions and charge_flat_fees)
	subID := ulid.Make().String()      // subscriptions
	subPhaseID := ulid.Make().String() // subscription_phases
	subItemID := ulid.Make().String()  // subscription_items

	flatFeeID := ulid.Make().String() // charge_flat_fees

	// organization_default_tax_codes: both FK columns pointing at group1Loser.
	orgDTCID := ulid.Make().String()

	runner{
		stops: stops{
			{
				// Stop 1: after migration 20260520130000 has been applied.
				// We insert all fixture rows here, before the dedup migration runs.
				version:   20260520130000,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Insert tax_codes rows.
					// Group 1 loser: non-system, created at T1.
					_, err := db.Exec(`
						INSERT INTO tax_codes (
							id, namespace, created_at, updated_at,
							name, key, app_mappings
						) VALUES (
							$1, $2, $3, $3,
							'Stripe txcd_10103001 (auto)', 'stripe_txcd_10103001',
							'[{"app_type":"stripe","tax_code":"txcd_10103001"}]'::jsonb
						)`, group1Loser, namespace, t1)
					require.NoError(t, err)

					// Group 1 winner: system-managed, created at T2 (newer, but is_system wins).
					_, err = db.Exec(`
						INSERT INTO tax_codes (
							id, namespace, created_at, updated_at,
							name, key, app_mappings, annotations
						) VALUES (
							$1, $2, $3, $3,
							'SaaS Business', 'saas_business',
							'[{"app_type":"stripe","tax_code":"txcd_10103001"}]'::jsonb,
							'{"managed_by":"system"}'::jsonb
						)`, group1Winner, namespace, t2)
					require.NoError(t, err)

					// Group 2 loser: non-system, created at T2 (newer → loses).
					_, err = db.Exec(`
						INSERT INTO tax_codes (
							id, namespace, created_at, updated_at,
							name, key, app_mappings
						) VALUES (
							$1, $2, $3, $3,
							'Stripe txcd_99999999 (auto)', 'stripe_txcd_99999999',
							'[{"app_type":"stripe","tax_code":"txcd_99999999"}]'::jsonb
						)`, group2Loser, namespace, t2)
					require.NoError(t, err)

					// Group 2 winner: non-system, created at T1 (older → wins).
					_, err = db.Exec(`
						INSERT INTO tax_codes (
							id, namespace, created_at, updated_at,
							name, key, app_mappings
						) VALUES (
							$1, $2, $3, $3,
							'Legacy 99', 'legacy_99',
							'[{"app_type":"stripe","tax_code":"txcd_99999999"}]'::jsonb
						)`, group2Winner, namespace, t1)
					require.NoError(t, err)

					// Singleton: no duplicate, must remain untouched.
					_, err = db.Exec(`
						INSERT INTO tax_codes (
							id, namespace, created_at, updated_at,
							name, key, app_mappings
						) VALUES (
							$1, $2, $3, $3,
							'Singleton Code', 'singleton_code',
							'[{"app_type":"stripe","tax_code":"txcd_unique_000"}]'::jsonb
						)`, singleton, namespace, t1)
					require.NoError(t, err)

					// billing_workflow_configs row pointing at group1Loser.
					_, err = db.Exec(`
						INSERT INTO billing_workflow_configs (
							id, namespace, created_at, updated_at,
							collection_alignment, line_collection_period,
							invoice_auto_advance, invoice_draft_period,
							invoice_due_after, invoice_collection_method,
							invoice_progressive_billing, tax_code_id
						) VALUES (
							$1, $2, NOW(), NOW(),
							'subscription', 'P1M',
							true, 'P1D',
							'P30D', 'charge_automatically',
							false, $3
						)`, wfcRowID, namespace, group1Loser)
					require.NoError(t, err)

					// plan → plan_phases → plan_rate_cards chain.
					_, err = db.Exec(`
						INSERT INTO plans (
							id, namespace, created_at, updated_at,
							name, key, version, currency,
							billing_cadence, pro_rating_config
						) VALUES (
							$1, $2, NOW(), NOW(),
							'Test Plan', 'test-plan', 1, 'USD',
							'P1M', '{"enabled":true,"mode":"prorate_prices"}'::jsonb
						)`, planID, namespace)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO plan_phases (
							id, namespace, created_at, updated_at,
							name, key, plan_id, index
						) VALUES (
							$1, $2, NOW(), NOW(),
							'Default Phase', 'default', $3, 0
						)`, phaseID, namespace, planID)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO plan_rate_cards (
							id, namespace, created_at, updated_at,
							name, key, type, phase_id, tax_code_id
						) VALUES (
							$1, $2, NOW(), NOW(),
							'Test Rate Card', 'test-rc', 'FLAT_FEE', $3, $4
						)`, prcRowID, namespace, phaseID, group1Loser)
					require.NoError(t, err)

					// customers → subscriptions → subscription_phases → subscription_items chain.
					_, err = db.Exec(`
						INSERT INTO customers (
							id, namespace, created_at, updated_at,
							key, name
						) VALUES (
							$1, $2, NOW(), NOW(),
							'dedupe-customer', 'Dedupe Customer'
						)`, customerID, namespace)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO subscriptions (
							id, namespace, created_at, updated_at,
							active_from, currency, customer_id, plan_id,
							billing_cadence, pro_rating_config, billing_anchor
						) VALUES (
							$1, $2, NOW(), NOW(),
							'2024-01-01 00:00:00', 'USD', $3, NULL,
							'P1M', '{"enabled":true,"mode":"prorate_prices"}'::jsonb,
							'2024-01-01 00:00:00'
						)`, subID, namespace, customerID)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO subscription_phases (
							id, namespace, created_at, updated_at,
							key, name, subscription_id, active_from
						) VALUES (
							$1, $2, NOW(), NOW(),
							'default', 'Default Phase', $3, '2024-01-01 00:00:00'
						)`, subPhaseID, namespace, subID)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO subscription_items (
							id, namespace, created_at, updated_at,
							active_from, key, name, phase_id, tax_code_id
						) VALUES (
							$1, $2, NOW(), NOW(),
							'2024-01-01 00:00:00', 'item-1', 'Item 1', $3, $4
						)`, subItemID, namespace, subPhaseID, group1Loser)
					require.NoError(t, err)

					// charge_flat_fees row pointing at group1Loser.
					_, err = db.Exec(`
						INSERT INTO charge_flat_fees (
							id, namespace,
							payment_term, invoice_at, settlement_mode,
							pro_rating, amount_before_proration, amount_after_proration,
							service_period_from, service_period_to,
							billing_period_from, billing_period_to,
							full_service_period_from, full_service_period_to,
							status, status_detailed, unique_reference_id, currency, managed_by,
							created_at, updated_at, name, customer_id,
							tax_code_id
						) VALUES (
							$1, $2,
							'in_advance', NOW(), 'invoice_only',
							'no_prorating', 100, 100,
							'2024-01-01 00:00:00', '2024-02-01 00:00:00',
							'2024-01-01 00:00:00', '2024-02-01 00:00:00',
							'2024-01-01 00:00:00', '2024-02-01 00:00:00',
							'final', 'final', 'dedup-flat-fee-ref', 'USD', 'subscription',
							NOW(), NOW(), 'Flat Fee Charge', $3,
							$4
						)`, flatFeeID, namespace, customerID, group1Loser)
					require.NoError(t, err)

					// organization_default_tax_codes: both columns pointing at group1Loser.
					_, err = db.Exec(`
						INSERT INTO organization_default_tax_codes (
							id, namespace, created_at, updated_at,
							invoicing_tax_code_id, credit_grant_tax_code_id
						) VALUES (
							$1, $2, NOW(), NOW(), $3, $3
						)`, orgDTCID, namespace, group1Loser)
					require.NoError(t, err)
				},
			},
			{
				// Stop 2: after the dedup migration 20260527120000 has run.
				// Assert all losers are soft-deleted, winners are live,
				// and every FK row now points at the winner.
				version:   20260527120000,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Group 1 loser must be soft-deleted.
					var deletedAt sql.NullTime
					err := db.QueryRow(`
						SELECT deleted_at FROM tax_codes WHERE id = $1
					`, group1Loser).Scan(&deletedAt)
					require.NoError(t, err)
					require.True(t, deletedAt.Valid, "group1Loser should be soft-deleted after dedup")

					// Group 1 winner must remain live.
					err = db.QueryRow(`
						SELECT deleted_at FROM tax_codes WHERE id = $1
					`, group1Winner).Scan(&deletedAt)
					require.NoError(t, err)
					require.False(t, deletedAt.Valid, "group1Winner should remain live after dedup")

					// Group 2 loser must be soft-deleted.
					err = db.QueryRow(`
						SELECT deleted_at FROM tax_codes WHERE id = $1
					`, group2Loser).Scan(&deletedAt)
					require.NoError(t, err)
					require.True(t, deletedAt.Valid, "group2Loser should be soft-deleted after dedup")

					// Group 2 winner must remain live.
					err = db.QueryRow(`
						SELECT deleted_at FROM tax_codes WHERE id = $1
					`, group2Winner).Scan(&deletedAt)
					require.NoError(t, err)
					require.False(t, deletedAt.Valid, "group2Winner should remain live after dedup")

					// Singleton must remain untouched (still live).
					err = db.QueryRow(`
						SELECT deleted_at FROM tax_codes WHERE id = $1
					`, singleton).Scan(&deletedAt)
					require.NoError(t, err)
					require.False(t, deletedAt.Valid, "singleton should not be touched by dedup")

					// billing_workflow_configs FK must now point at group1Winner.
					var taxCodeID string
					err = db.QueryRow(`
						SELECT tax_code_id FROM billing_workflow_configs WHERE id = $1
					`, wfcRowID).Scan(&taxCodeID)
					require.NoError(t, err)
					require.Equal(t, group1Winner, taxCodeID, "billing_workflow_configs.tax_code_id should be repointed to group1Winner")

					// plan_rate_cards FK must now point at group1Winner.
					err = db.QueryRow(`
						SELECT tax_code_id FROM plan_rate_cards WHERE id = $1
					`, prcRowID).Scan(&taxCodeID)
					require.NoError(t, err)
					require.Equal(t, group1Winner, taxCodeID, "plan_rate_cards.tax_code_id should be repointed to group1Winner")

					// subscription_items FK must now point at group1Winner.
					err = db.QueryRow(`
						SELECT tax_code_id FROM subscription_items WHERE id = $1
					`, subItemID).Scan(&taxCodeID)
					require.NoError(t, err)
					require.Equal(t, group1Winner, taxCodeID, "subscription_items.tax_code_id should be repointed to group1Winner")

					// charge_flat_fees FK must now point at group1Winner.
					err = db.QueryRow(`
						SELECT tax_code_id FROM charge_flat_fees WHERE id = $1
					`, flatFeeID).Scan(&taxCodeID)
					require.NoError(t, err)
					require.Equal(t, group1Winner, taxCodeID, "charge_flat_fees.tax_code_id should be repointed to group1Winner")

					// organization_default_tax_codes: both columns must point at group1Winner.
					var invTaxCodeID, cgTaxCodeID string
					err = db.QueryRow(`
						SELECT invoicing_tax_code_id, credit_grant_tax_code_id
						FROM organization_default_tax_codes WHERE id = $1
					`, orgDTCID).Scan(&invTaxCodeID, &cgTaxCodeID)
					require.NoError(t, err)
					require.Equal(t, group1Winner, invTaxCodeID, "organization_default_tax_codes.invoicing_tax_code_id should be repointed to group1Winner")
					require.Equal(t, group1Winner, cgTaxCodeID, "organization_default_tax_codes.credit_grant_tax_code_id should be repointed to group1Winner")
				},
			},
		},
	}.Test(t)
}
