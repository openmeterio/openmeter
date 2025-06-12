package migrate_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func TestStartAfterChange(t *testing.T) {
	// This is an example test adding a plan phase before start_after is changed to duration
	// and asserting in the next step that it is in fact deleted as per the migration
	runner{
		stops: stops{
			{
				// before: 20241230152834_app_stripe_account_id_unique.up.sql
				// after: 20250103121359_plan-phase-duration.up.sql
				version:   20241230152834,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// As we're changing the phase duration format while there are existing plans present lets make sure thats not an issue
					someUlid := ulid.Make()
					_, err := db.Exec(`
					INSERT INTO plans (namespace, id, name, key, version, created_at, updated_at)
					VALUES ('default', $1, 'test', 'test', 1, NOW(), NOW())`,
						someUlid.String(),
					)
					require.NoError(t, err)

					_, err = db.Exec(`
					INSERT INTO plan_phases (namespace, id, plan_id, name, key, start_after, created_at, updated_at)
					VALUES ('default', $1, $2, 'default', 'default', 'P1M', NOW(), NOW())`,
						ulid.Make().String(), someUlid.String(),
					)
					require.NoError(t, err)
				},
			},
			{
				// The version at which all phases are deleted: 20250103121359_plan-phase-duration.up.sql
				version:   20250103121359,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					var count int

					res := db.QueryRow(`SELECT COUNT(*) FROM plan_phases`)
					require.NoError(t, res.Scan(&count))
					require.Equal(t, 0, count)

					res = db.QueryRow(`SELECT COUNT(*) FROM plans`)
					require.NoError(t, res.Scan(&count))
					require.Equal(t, 1, count)
				},
			},
		},
	}.Test(t)
}

func TestEntitlementISO(t *testing.T) {
	keptEntId := ulid.Make()
	lostEntId := ulid.Make()

	keptGrantId := ulid.Make()
	lostGrantId := ulid.Make()

	runner{
		stops: stops{
			{
				// before: 20250108103427_billing-profile-progressive-billing-flag.up.sql
				// after: 20250109231835_entitlements-usage-period-iso.up.sql
				version:   20250108103427,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// We need to set up a feature
					featId := ulid.Make()
					_, err := db.Exec(`
					INSERT INTO features (namespace, id, key, name, created_at, updated_at)
					VALUES ('default', $1, 'feat_1', 'feat 1', NOW(), NOW())`,
						featId.String(),
					)
					require.NoError(t, err)

					// Kept
					// We're chancing the duration format in entitlements and credits
					_, err = db.Exec(`
					INSERT INTO entitlements (namespace, id, created_at, updated_at, entitlement_type, feature_key, feature_id, subject_key, usage_period_interval)
					VALUES ('default', $1, '2025-01-08 23:18:35', NOW(), 'BOOL', 'feature_1', $2, 'subject_1', 'MONTH')`,
						keptEntId.String(),
						featId.String(),
					)
					require.NoError(t, err)

					_, err = db.Exec(`
					INSERT INTO grants (namespace, id, owner_id, created_at, updated_at, recurrence_period, amount, effective_at, expiration, expires_at, reset_max_rollover, reset_min_rollover)
					VALUES ('default', $1, $2, NOW(), NOW(), 'MONTH', 0, NOW(), '{}'::jsonb, NOW(), 0, 0)`,
						keptGrantId.String(), keptEntId.String(),
					)
					require.NoError(t, err)

					// Lost
					// We're changing the duration format in entitlements and credits
					_, err = db.Exec(`
					INSERT INTO entitlements (namespace, id, created_at, updated_at, entitlement_type, feature_key, feature_id, subject_key, usage_period_interval)
					VALUES ('default', $1, '2025-02-12 23:18:35', NOW(), 'BOOL', 'feature_1', $2, 'subject_1', 'P2W')`,
						lostEntId.String(),
						featId.String(),
					)
					require.NoError(t, err)

					_, err = db.Exec(`
					INSERT INTO grants (namespace, id, owner_id, created_at, updated_at, recurrence_period, amount, effective_at, expiration, expires_at, reset_max_rollover, reset_min_rollover)
					VALUES ('default', $1, $2, NOW(), NOW(), 'P2W', 0, NOW(), '{}'::jsonb, NOW(), 0, 0)`,
						lostGrantId.String(), lostEntId.String(),
					)
					require.NoError(t, err)
				},
			},
			{
				// 20250109231835_entitlements-usage-period-iso.up.sql
				version:   20250109231835,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Let's validate that the usage_period_interval has been changed to iso format
					var usagePeriodInterval string

					res := db.QueryRow(`SELECT usage_period_interval FROM entitlements WHERE id = $1`, keptEntId.String())
					require.NoError(t, res.Scan(&usagePeriodInterval))

					require.Equal(t, "P1M", usagePeriodInterval)

					// Let's validate that the grants table has been updated as well
					var recurrencePeriod string

					res = db.QueryRow(`SELECT recurrence_period FROM grants WHERE id = $1`, keptGrantId.String())
					require.NoError(t, res.Scan(&recurrencePeriod))

					require.Equal(t, "P1M", recurrencePeriod)
				},
			},
			{
				// after 20250109231835_entitlements-usage-period-iso.up.sql
				version:   20250108103427,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					// Let's check that deleted_at is set for both the entitlements and grants table

					var deletedAt sql.NullString

					// Let's check it was kept
					res := db.QueryRow(`SELECT deleted_at FROM entitlements WHERE id = $1`, keptEntId.String())
					require.NoError(t, res.Scan(&deletedAt))

					require.Empty(t, deletedAt)

					res = db.QueryRow(`SELECT deleted_at FROM grants WHERE id = $1`, keptGrantId.String())
					require.NoError(t, res.Scan(&deletedAt))

					require.Empty(t, deletedAt)

					// Let's check it was lost
					res = db.QueryRow(`SELECT deleted_at FROM entitlements WHERE id = $1`, lostEntId.String())
					require.NoError(t, res.Scan(&deletedAt))

					require.NotEmpty(t, deletedAt)

					res = db.QueryRow(`SELECT deleted_at FROM grants WHERE id = $1`, lostGrantId.String())
					require.NoError(t, res.Scan(&deletedAt))

					require.NotEmpty(t, deletedAt)
				},
			},
		},
	}.Test(t)
}

func TestPlanBillingCadenceProRatingMigration(t *testing.T) {
	// Plan with multiple rate cards in last phase - should pick shortest billing cadence
	plan1Id := ulid.Make()
	plan1Phase1Id := ulid.Make()
	plan1Phase2Id := ulid.Make()
	plan1RC1Id := ulid.Make()
	plan1RC2Id := ulid.Make()
	plan1RC3Id := ulid.Make()
	plan1RC4Id := ulid.Make()

	// Plan with multiple rate cards in last phase
	plan2Id := ulid.Make()
	plan2PhaseId := ulid.Make()
	plan2RC1Id := ulid.Make()
	plan2RC2Id := ulid.Make()

	// Plan without rate cards - should get default
	plan3Id := ulid.Make()
	plan3PhaseId := ulid.Make()

	runner{
		stops: stops{
			{
				// TEST SCENARIO 1: Setup test data before the migration
				// Creates three plans with different rate card configurations:
				// - Plan 1: Multiple phases with multiple rate cards in last phase (P1W, P1M, P3M, P1Y)
				//   Expected: Should pick shortest cadence P1W from last phase
				// - Plan 2: Single phase with multiple rate cards (P2W, P6M)
				//   Expected: Should pick shortest cadence P2W from last phase
				// - Plan 3: Single phase with no rate cards
				//   Expected: Should get default cadence P1M
				//
				// Before: 20250609204117_billing-migrate-split-line-groups.up.sql
				// After: 20250610101736_plan-subscription-billing-cadence.up.sql
				version:   20250609204117,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Create plan 1 with multiple phases and rate cards
					_, err := db.Exec(`
					INSERT INTO plans (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						version,
						name,
						currency
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'multi_cadence_plan',
						1,
						'Multi Cadence Plan',
						'USD'
					)`,
						plan1Id.String(),
					)
					require.NoError(t, err)

					// Create first phase (trial) with P1M duration
					_, err = db.Exec(`
					INSERT INTO plan_phases (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						plan_id,
						index,
						duration
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'trial',
						'Trial Phase',
						$2,
						0,
						'P1M'
					)`,
						plan1Phase1Id.String(),
						plan1Id.String(),
					)
					require.NoError(t, err)

					// Create last phase (pro) with NULL duration (infinite)
					_, err = db.Exec(`
					INSERT INTO plan_phases (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						plan_id,
						index,
						duration
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'pro',
						'Pro Phase',
						$2,
						1,
						NULL
					)`,
						plan1Phase2Id.String(),
						plan1Id.String(),
					)
					require.NoError(t, err)

					// Create rate cards in last phase with different billing cadences
					// Since last phase has NULL duration, rate cards can have any billing cadence

					// Weekly rate card (shortest - should be picked)
					_, err = db.Exec(`
					INSERT INTO plan_rate_cards (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						phase_id,
						type,
						billing_cadence
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'weekly_rc',
						'Weekly Rate Card',
						$2,
						'FLAT_FEE',
						'P1W'
					)`,
						plan1RC1Id.String(),
						plan1Phase2Id.String(),
					)
					require.NoError(t, err)

					// Monthly rate card
					_, err = db.Exec(`
					INSERT INTO plan_rate_cards (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						phase_id,
						type,
						billing_cadence
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'monthly_rc',
						'Monthly Rate Card',
						$2,
						'FLAT_FEE',
						'P1M'
					)`,
						plan1RC2Id.String(),
						plan1Phase2Id.String(),
					)
					require.NoError(t, err)

					// Quarterly rate card
					_, err = db.Exec(`
					INSERT INTO plan_rate_cards (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						phase_id,
						type,
						billing_cadence
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'quarterly_rc',
						'Quarterly Rate Card',
						$2,
						'USAGE_BASED',
						'P3M'
					)`,
						plan1RC3Id.String(),
						plan1Phase2Id.String(),
					)
					require.NoError(t, err)

					// Yearly rate card
					_, err = db.Exec(`
					INSERT INTO plan_rate_cards (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						phase_id,
						type,
						billing_cadence
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'yearly_rc',
						'Yearly Rate Card',
						$2,
						'FLAT_FEE',
						'P1Y'
					)`,
						plan1RC4Id.String(),
						plan1Phase2Id.String(),
					)
					require.NoError(t, err)

					// Create plan 2 with multiple rate cards in last phase
					_, err = db.Exec(`
					INSERT INTO plans (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						version,
						name,
						currency
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'multiple_rc_plan',
						1,
						'Multiple Rate Card Plan',
						'USD'
					)`,
						plan2Id.String(),
					)
					require.NoError(t, err)

					// Create last phase for plan 2 (NULL duration)
					_, err = db.Exec(`
					INSERT INTO plan_phases (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						plan_id,
						index,
						duration
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'main',
						'Main Phase',
						$2,
						0,
						NULL
					)`,
						plan2PhaseId.String(),
						plan2Id.String(),
					)
					require.NoError(t, err)

					// Create rate cards with different billing cadences
					// Bi-weekly rate card (shortest - should be picked)
					_, err = db.Exec(`
					INSERT INTO plan_rate_cards (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						phase_id,
						type,
						billing_cadence
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'biweekly_rc',
						'Bi-weekly Rate Card',
						$2,
						'USAGE_BASED',
						'P2W'
					)`,
						plan2RC1Id.String(),
						plan2PhaseId.String(),
					)
					require.NoError(t, err)

					// Semi-annual rate card
					_, err = db.Exec(`
					INSERT INTO plan_rate_cards (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						phase_id,
						type,
						billing_cadence
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'semiannual_rc',
						'Semi-annual Rate Card',
						$2,
						'FLAT_FEE',
						'P6M'
					)`,
						plan2RC2Id.String(),
						plan2PhaseId.String(),
					)
					require.NoError(t, err)

					// Create plan 3 without rate cards
					_, err = db.Exec(`
					INSERT INTO plans (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						version,
						name,
						currency
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'no_ratecard_plan',
						1,
						'No Rate Card Plan',
						'USD'
					)`,
						plan3Id.String(),
					)
					require.NoError(t, err)

					// Create last phase for plan 3 (no rate cards)
					_, err = db.Exec(`
					INSERT INTO plan_phases (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						plan_id,
						index,
						duration
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'empty',
						'Empty Phase',
						$2,
						0,
						NULL
					)`,
						plan3PhaseId.String(),
						plan3Id.String(),
					)
					require.NoError(t, err)
				},
			},
			{
				// TEST SCENARIO 2: Verify migration results and PostgreSQL interval casting
				// This test verifies two critical aspects:
				// 1. PostgreSQL's ability to cast ISO8601 duration strings to intervals and compare them
				//    - Validates that 'P1W'::interval < 'P1M'::interval < 'P3M'::interval < 'P1Y'::interval
				//    - Confirms the ORDER BY billing_cadence::interval ASC logic works correctly
				// 2. Migration correctly applied the business logic:
				//    - Plan 1: Should have billing_cadence='P1W' (shortest from last phase rate cards)
				//    - Plan 2: Should have billing_cadence='P2W' (shortest from last phase rate cards)
				//    - Plan 3: Should have billing_cadence='P1M' (default fallback for plans without rate cards)
				//    - All plans: Should have pro_rating_config='{"enabled":true,"mode":"prorate_prices"}'
				//
				// Before: 20250609204117_billing-migrate-split-line-groups.up.sql
				// After: 20250610101736_plan-subscription-billing-cadence.up.sql
				version:   20250610101736,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// First, verify that PostgreSQL can cast billing_cadence to interval and compare them
					// This tests the core functionality used in the migration
					rows, err := db.Query(`
					SELECT prc.billing_cadence, prc.billing_cadence::interval
					FROM plan_rate_cards prc
					JOIN plan_phases pp ON prc.phase_id = pp.id
					WHERE pp.plan_id = $1 AND pp.duration IS NULL
					ORDER BY prc.billing_cadence::interval ASC
				`, plan1Id.String())
					require.NoError(t, err)
					defer rows.Close()

					var cadences []string
					var intervals []string
					for rows.Next() {
						var cadence, interval string
						err := rows.Scan(&cadence, &interval)
						require.NoError(t, err)
						cadences = append(cadences, cadence)
						intervals = append(intervals, interval)
					}

					// Verify that the cadences are in the expected order (shortest to longest)
					require.Equal(t, []string{"P1W", "P1M", "P3M", "P1Y"}, cadences,
						"Billing cadences should be ordered from shortest to longest")

					// Verify that PostgreSQL correctly converted ISO8601 durations to intervals
					expectedIntervals := []string{"7 days", "1 mon", "3 mons", "1 year"}
					require.Equal(t, expectedIntervals, intervals,
						"PostgreSQL should correctly convert ISO8601 durations to intervals")

					// Now verify that the migration used this logic correctly
					// Check plan 1 - should have shortest billing cadence (P1W)
					var billingCadence string
					var proRatingConfigJSON string
					err = db.QueryRow(`
					SELECT billing_cadence, pro_rating_config::text
					FROM plans
					WHERE id = $1
				`, plan1Id.String()).Scan(&billingCadence, &proRatingConfigJSON)
					require.NoError(t, err)

					require.Equal(t, "P1W", billingCadence, "Plan 1 should have shortest billing cadence (weekly) from last phase")

					// Validate pro-rating config
					var proRatingConfig productcatalog.ProRatingConfig
					err = json.Unmarshal([]byte(proRatingConfigJSON), &proRatingConfig)
					require.NoError(t, err)
					require.True(t, proRatingConfig.Enabled)
					require.Equal(t, productcatalog.ProRatingModeProratePrices, proRatingConfig.Mode)

					// Check plan 2 - should have the shortest billing cadence (P2W)
					err = db.QueryRow(`
					SELECT billing_cadence, pro_rating_config::text
					FROM plans
					WHERE id = $1
				`, plan2Id.String()).Scan(&billingCadence, &proRatingConfigJSON)
					require.NoError(t, err)

					require.Equal(t, "P2W", billingCadence, "Plan 2 should have shortest billing cadence (bi-weekly)")

					// Validate pro-rating config
					err = json.Unmarshal([]byte(proRatingConfigJSON), &proRatingConfig)
					require.NoError(t, err)
					require.True(t, proRatingConfig.Enabled)
					require.Equal(t, productcatalog.ProRatingModeProratePrices, proRatingConfig.Mode)

					// Check plan 3 - should have default billing cadence (P1M)
					err = db.QueryRow(`
					SELECT billing_cadence, pro_rating_config::text
					FROM plans
					WHERE id = $1
				`, plan3Id.String()).Scan(&billingCadence, &proRatingConfigJSON)
					require.NoError(t, err)

					require.Equal(t, "P1M", billingCadence, "Plan 3 should have default billing cadence")

					// Validate pro-rating config
					err = json.Unmarshal([]byte(proRatingConfigJSON), &proRatingConfig)
					require.NoError(t, err)
					require.True(t, proRatingConfig.Enabled)
					require.Equal(t, productcatalog.ProRatingModeProratePrices, proRatingConfig.Mode)

					// Additional verification: Test that P2W < P6M for plan 2
					var comparison bool
					err = db.QueryRow(`SELECT 'P2W'::interval < 'P6M'::interval`).Scan(&comparison)
					require.NoError(t, err)
					require.True(t, comparison, "P2W should be less than P6M when cast to intervals")
				},
			},
			{
				// TEST SCENARIO 3: Verify down migration removes added columns
				// This test ensures the down migration properly cleans up:
				// - Removes billing_cadence column from plans table
				// - Removes pro_rating_config column from plans table
				// - Verifies no residual schema artifacts remain
				// This confirms the migration is fully reversible without leaving orphaned columns.
				//
				// Before: 20250606115859_subject_namespace.up.sql
				// After: 20250606130010_plan-subscription-billing-cadence.up.sql
				version:   20250606115859,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					// Check that billing_cadence and pro_rating_config columns no longer exist
					rows, err := db.Query(`
					SELECT column_name
					FROM information_schema.columns
					WHERE table_name = 'plans'
					AND column_name IN ('billing_cadence', 'pro_rating_config')
				`)
					require.NoError(t, err)
					defer rows.Close()

					var foundColumns []string
					for rows.Next() {
						var columnName string
						err := rows.Scan(&columnName)
						require.NoError(t, err)
						foundColumns = append(foundColumns, columnName)
					}

					require.Empty(t, foundColumns, "billing_cadence and pro_rating_config columns should not exist after down migration")
				},
			},
		},
	}.Test(t)
}

func TestSubscriptionBillingCadenceProRatingMigration(t *testing.T) {
	// Subscription with multiple phases and items with different billing cadences
	sub1Id := ulid.Make()
	sub1Phase1Id := ulid.Make()
	sub1Phase2Id := ulid.Make()
	sub1Item1Id := ulid.Make()
	sub1Item2Id := ulid.Make()
	sub1Item3Id := ulid.Make()

	// Subscription with single phase and items
	sub2Id := ulid.Make()
	sub2PhaseId := ulid.Make()
	sub2Item1Id := ulid.Make()
	sub2Item2Id := ulid.Make()

	// Subscription without items - should get default
	sub3Id := ulid.Make()
	sub3PhaseId := ulid.Make()

	runner{
		stops: stops{
			{
				// TEST SCENARIO 1: Setup test data before the migration
				// Creates three subscriptions with different subscription item configurations:
				// - Subscription 1: Multiple phases with multiple items in last phase (P1W, P1M, P3M)
				//   Expected: Should pick shortest cadence P1W from last phase items
				// - Subscription 2: Single phase with multiple items (P2W, P6M)
				//   Expected: Should pick shortest cadence P2W from items
				// - Subscription 3: Single phase with no items
				//   Expected: Should get default cadence P1M
				//
				// Before: 20250606115859_subject_namespace.up.sql
				// After: 20250606130010_plan-subscription-billing-cadence.up.sql
				version:   20250606115859,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Create customers first (required for foreign key constraint)
					for i := 1; i <= 3; i++ {
						_, err := db.Exec(`
						INSERT INTO customers (
							namespace,
							id,
							key,
							created_at,
							updated_at,
							name,
							primary_email,
							currency
						)
						VALUES (
							'default',
							$1,
							$2,
							NOW(),
							NOW(),
							$3,
							$4,
							'USD'
						)`,
							fmt.Sprintf("customer_%d", i),
							fmt.Sprintf("customer_key_%d", i),
							fmt.Sprintf("Customer %d", i),
							fmt.Sprintf("customer%d@example.com", i),
						)
						require.NoError(t, err)
					}

					// Create subscription 1 with multiple phases and items
					_, err := db.Exec(`
					INSERT INTO subscriptions (
						namespace,
						id,
						created_at,
						updated_at,
						active_from,
						plan_id,
						customer_id,
						currency
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'2024-01-01 00:00:00',
						NULL,
						'customer_1',
						'USD'
					)`,
						sub1Id.String(),
					)
					require.NoError(t, err)

					// Create first phase (trial)
					_, err = db.Exec(`
					INSERT INTO subscription_phases (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						subscription_id,
						active_from
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'trial',
						'Trial Phase',
						$2,
						'2024-01-01 00:00:00'
					)`,
						sub1Phase1Id.String(),
						sub1Id.String(),
					)
					require.NoError(t, err)

					// Create last phase (pro) - most recent active_from
					_, err = db.Exec(`
					INSERT INTO subscription_phases (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						subscription_id,
						active_from
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'pro',
						'Pro Phase',
						$2,
						'2024-02-01 00:00:00'
					)`,
						sub1Phase2Id.String(),
						sub1Id.String(),
					)
					require.NoError(t, err)

					// Create subscription items in last phase with different billing cadences
					// Weekly item (shortest - should be picked)
					_, err = db.Exec(`
					INSERT INTO subscription_items (
						namespace,
						id,
						created_at,
						updated_at,
						active_from,
						key,
						name,
						phase_id,
						billing_cadence
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'2024-02-01 00:00:00',
						'weekly_item',
						'Weekly Item',
						$2,
						'P1W'
					)`,
						sub1Item1Id.String(),
						sub1Phase2Id.String(),
					)
					require.NoError(t, err)

					// Monthly item
					_, err = db.Exec(`
					INSERT INTO subscription_items (
						namespace,
						id,
						created_at,
						updated_at,
						active_from,
						key,
						name,
						phase_id,
						billing_cadence
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'2024-02-01 00:00:00',
						'monthly_item',
						'Monthly Item',
						$2,
						'P1M'
					)`,
						sub1Item2Id.String(),
						sub1Phase2Id.String(),
					)
					require.NoError(t, err)

					// Quarterly item
					_, err = db.Exec(`
					INSERT INTO subscription_items (
						namespace,
						id,
						created_at,
						updated_at,
						active_from,
						key,
						name,
						phase_id,
						billing_cadence
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'2024-02-01 00:00:00',
						'quarterly_item',
						'Quarterly Item',
						$2,
						'P3M'
					)`,
						sub1Item3Id.String(),
						sub1Phase2Id.String(),
					)
					require.NoError(t, err)

					// Create subscription 2 with single phase and multiple items
					_, err = db.Exec(`
					INSERT INTO subscriptions (
						namespace,
						id,
						created_at,
						updated_at,
						active_from,
						plan_id,
						customer_id,
						currency
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'2024-01-01 00:00:00',
						NULL,
						'customer_2',
						'USD'
					)`,
						sub2Id.String(),
					)
					require.NoError(t, err)

					// Create phase for subscription 2
					_, err = db.Exec(`
					INSERT INTO subscription_phases (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						subscription_id,
						active_from
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'main',
						'Main Phase',
						$2,
						'2024-01-01 00:00:00'
					)`,
						sub2PhaseId.String(),
						sub2Id.String(),
					)
					require.NoError(t, err)

					// Create items with different billing cadences
					// Bi-weekly item (shortest - should be picked)
					_, err = db.Exec(`
					INSERT INTO subscription_items (
						namespace,
						id,
						created_at,
						updated_at,
						active_from,
						key,
						name,
						phase_id,
						billing_cadence
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'2024-01-01 00:00:00',
						'biweekly_item',
						'Bi-weekly Item',
						$2,
						'P2W'
					)`,
						sub2Item1Id.String(),
						sub2PhaseId.String(),
					)
					require.NoError(t, err)

					// Semi-annual item
					_, err = db.Exec(`
					INSERT INTO subscription_items (
						namespace,
						id,
						created_at,
						updated_at,
						active_from,
						key,
						name,
						phase_id,
						billing_cadence
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'2024-01-01 00:00:00',
						'semiannual_item',
						'Semi-annual Item',
						$2,
						'P6M'
					)`,
						sub2Item2Id.String(),
						sub2PhaseId.String(),
					)
					require.NoError(t, err)

					// Create subscription 3 without items
					_, err = db.Exec(`
					INSERT INTO subscriptions (
						namespace,
						id,
						created_at,
						updated_at,
						active_from,
						plan_id,
						customer_id,
						currency
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'2024-01-01 00:00:00',
						NULL,
						'customer_3',
						'USD'
					)`,
						sub3Id.String(),
					)
					require.NoError(t, err)

					// Create phase for subscription 3 (no items)
					_, err = db.Exec(`
					INSERT INTO subscription_phases (
						namespace,
						id,
						created_at,
						updated_at,
						key,
						name,
						subscription_id,
						active_from
					)
					VALUES (
						'default',
						$1,
						NOW(),
						NOW(),
						'empty',
						'Empty Phase',
						$2,
						'2024-01-01 00:00:00'
					)`,
						sub3PhaseId.String(),
						sub3Id.String(),
					)
					require.NoError(t, err)
				},
			},
			{
				// TEST SCENARIO 2: Verify migration results and PostgreSQL interval casting for subscriptions
				// This test verifies:
				// 1. PostgreSQL's ability to cast ISO8601 duration strings to intervals for subscription items
				// 2. Migration correctly applied the business logic:
				//    - Subscription 1: Should have billing_cadence='P1W' (shortest from last phase items)
				//    - Subscription 2: Should have billing_cadence='P2W' (shortest from items)
				//    - Subscription 3: Should have billing_cadence='P1M' (default fallback)
				//    - All subscriptions: Should have pro_rating_config='{"enabled":true,"mode":"prorate_prices"}'
				//
				// Before: 20250609204117_billing-migrate-split-line-groups.up.sql
				// After: 20250610101736_plan-subscription-billing-cadence.up.sql
				version:   20250610101736,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// First, verify that PostgreSQL can cast billing_cadence to interval for subscription items
					rows, err := db.Query(`
					SELECT si.billing_cadence, si.billing_cadence::interval
					FROM subscription_items si
					JOIN subscription_phases sp ON si.phase_id = sp.id
					WHERE sp.subscription_id = $1
					ORDER BY si.billing_cadence::interval ASC
				`, sub1Id.String())
					require.NoError(t, err)
					defer rows.Close()

					var cadences []string
					var intervals []string
					for rows.Next() {
						var cadence, interval string
						err := rows.Scan(&cadence, &interval)
						require.NoError(t, err)
						cadences = append(cadences, cadence)
						intervals = append(intervals, interval)
					}

					// Verify that the cadences are in the expected order (shortest to longest)
					require.Equal(t, []string{"P1W", "P1M", "P3M"}, cadences,
						"Subscription item billing cadences should be ordered from shortest to longest")

					// Verify that PostgreSQL correctly converted ISO8601 durations to intervals
					expectedIntervals := []string{"7 days", "1 mon", "3 mons"}
					require.Equal(t, expectedIntervals, intervals,
						"PostgreSQL should correctly convert ISO8601 durations to intervals for subscription items")

					// Now verify that the migration used this logic correctly
					// Check subscription 1 - should have shortest billing cadence (P1W)
					var billingCadence string
					var proRatingConfigJSON string
					err = db.QueryRow(`
					SELECT billing_cadence, pro_rating_config::text
					FROM subscriptions
					WHERE id = $1
				`, sub1Id.String()).Scan(&billingCadence, &proRatingConfigJSON)
					require.NoError(t, err)

					require.Equal(t, "P1W", billingCadence, "Subscription 1 should have shortest billing cadence (weekly) from last phase items")

					// Validate pro-rating config
					var proRatingConfig productcatalog.ProRatingConfig
					err = json.Unmarshal([]byte(proRatingConfigJSON), &proRatingConfig)
					require.NoError(t, err)
					require.True(t, proRatingConfig.Enabled)
					require.Equal(t, productcatalog.ProRatingModeProratePrices, proRatingConfig.Mode)

					// Check subscription 2 - should have the shortest billing cadence (P2W)
					err = db.QueryRow(`
					SELECT billing_cadence, pro_rating_config::text
					FROM subscriptions
					WHERE id = $1
				`, sub2Id.String()).Scan(&billingCadence, &proRatingConfigJSON)
					require.NoError(t, err)

					require.Equal(t, "P2W", billingCadence, "Subscription 2 should have shortest billing cadence (bi-weekly)")

					// Validate pro-rating config
					err = json.Unmarshal([]byte(proRatingConfigJSON), &proRatingConfig)
					require.NoError(t, err)
					require.True(t, proRatingConfig.Enabled)
					require.Equal(t, productcatalog.ProRatingModeProratePrices, proRatingConfig.Mode)

					// Check subscription 3 - should have default billing cadence (P1M)
					err = db.QueryRow(`
					SELECT billing_cadence, pro_rating_config::text
					FROM subscriptions
					WHERE id = $1
				`, sub3Id.String()).Scan(&billingCadence, &proRatingConfigJSON)
					require.NoError(t, err)

					require.Equal(t, "P1M", billingCadence, "Subscription 3 should have default billing cadence")

					// Validate pro-rating config
					err = json.Unmarshal([]byte(proRatingConfigJSON), &proRatingConfig)
					require.NoError(t, err)
					require.True(t, proRatingConfig.Enabled)
					require.Equal(t, productcatalog.ProRatingModeProratePrices, proRatingConfig.Mode)

					// Additional verification: Test that P2W < P6M for subscription 2
					var comparison bool
					err = db.QueryRow(`SELECT 'P2W'::interval < 'P6M'::interval`).Scan(&comparison)
					require.NoError(t, err)
					require.True(t, comparison, "P2W should be less than P6M when cast to intervals")
				},
			},
			{
				// TEST SCENARIO 3: Verify down migration removes added columns from subscriptions table
				// This test ensures the down migration properly cleans up:
				// - Removes billing_cadence column from subscriptions table
				// - Removes pro_rating_config column from subscriptions table
				// - Verifies no residual schema artifacts remain
				// This confirms the migration is fully reversible without leaving orphaned columns.
				//
				// Before: 20250609204117_billing-migrate-split-line-groups.up.sql
				// After: 20250610101736_plan-subscription-billing-cadence.up.sql
				version:   20250609204117,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					// Check that billing_cadence and pro_rating_config columns no longer exist in subscriptions table
					rows, err := db.Query(`
					SELECT column_name
					FROM information_schema.columns
					WHERE table_name = 'subscriptions'
					AND column_name IN ('billing_cadence', 'pro_rating_config')
				`)
					require.NoError(t, err)
					defer rows.Close()

					var foundColumns []string
					for rows.Next() {
						var columnName string
						err := rows.Scan(&columnName)
						require.NoError(t, err)
						foundColumns = append(foundColumns, columnName)
					}

					require.Empty(t, foundColumns, "billing_cadence and pro_rating_config columns should not exist in subscriptions table after down migration")
				},
			},
		},
	}.Test(t)
}
