package migrate_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func TestUsageResetAnchorTimesMigration(t *testing.T) {
	entId1 := ulid.Make()
	ent1UPAnchor := testutils.GetRFC3339Time(t, "2025-02-01T00:00:00Z")
	ent1ur1Id := ulid.Make()
	ent1ur2Id := ulid.Make()

	runner{stops{
		{
			// before: 20250218161614_billing-profile-fix-constraint.up.sql
			// after: 20250220150245_usage-reset-anchor-times.up.sql
			version:   20250218161614,
			direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				// 1. We need to set up a feature
				featId := ulid.Make()
				_, err := db.Exec(`
					INSERT INTO features (
						namespace,
						id,
						key,
						name,
						created_at,
						updated_at
					)
					VALUES (
						'default',
						$1,
						'feat_1',
						'feat 1',
						NOW(), -- don't mind that this is later than entitlement creation time, the DB doesn't care about it
						NOW()  -- don't mind that this is later than entitlement creation time, the DB doesn't care about it
					)`,
					featId.String(),
				)
				require.NoError(t, err)

				// 2. We need to set up an entitlement
				_, err = db.Exec(`
					INSERT INTO entitlements (
						namespace,
						id,
						created_at,
						updated_at,
						entitlement_type,
						feature_key,
						feature_id,
						subject_key,
						usage_period_interval,
						usage_period_anchor
					)
					VALUES (
						'default',
						$1,
						'2025-02-01 23:18:35',
						NOW(),
						'METERED',
						'feature_1',
						$2,
						'subject_1',
						'MONTH',
						$3
					)`,
					entId1.String(),
					featId.String(),
					ent1UPAnchor,
				)
				require.NoError(t, err)

				// 3. We need to set up 2 (so it can correctly choose the last one) past usage resets that show some usage
				_, err = db.Exec(`
					INSERT INTO usage_resets (namespace, id, created_at, updated_at, entitlement_id, reset_time)
					VALUES ('default', $1, NOW(), NOW(), $2, '2025-02-05T12:00:00Z')`,
					ent1ur1Id.String(),
					entId1.String(),
				)
				require.NoError(t, err)

				_, err = db.Exec(`
					INSERT INTO usage_resets (namespace, id, created_at, updated_at, entitlement_id, reset_time)
					VALUES ('default', $1, NOW(), NOW(), $2, '2025-02-011T12:00:00Z')`,
					ent1ur2Id.String(),
					entId1.String(),
				)
				require.NoError(t, err)
			},
		},
		{
			// Now we assert that the migration was successful
			version:   20250220150245,
			direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				// Let's assert that for for the first usage-reset (as its not the latest) the anchor is set to the reset time of the usage_reset
				var anchor time.Time
				var resetTime time.Time
				err := db.QueryRow(`SELECT anchor, reset_time FROM usage_resets WHERE id = $1`, ent1ur1Id.String()).Scan(&anchor, &resetTime)
				require.NoError(t, err)
				require.Equal(t, resetTime, anchor)

				// Let's assert that the second usage-reset was updated with the entitlement's usage_period_anchor
				var anchor2 time.Time
				err = db.QueryRow(`SELECT anchor FROM usage_resets WHERE id = $1`, ent1ur2Id.String()).Scan(&anchor2)
				require.NoError(t, err)
				require.Equal(t, ent1UPAnchor, anchor2.UTC())
			},
		},
	}}.Test(t)
}
