package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestStartAfterChange(t *testing.T) {
	// This is an example test adding a plan phase before start_after is changed to duration
	// and asserting in the next step that it is in fact deleted as per the migration
	runner{stops{
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
	}}.Test(t)
}

func TestEntitlementISO(t *testing.T) {
	keptEntId := ulid.Make()
	lostEntId := ulid.Make()

	keptGrantId := ulid.Make()
	lostGrantId := ulid.Make()

	runner{stops{
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
	}}.Test(t)
}
