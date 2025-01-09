package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func init() {
	// This is an example test adding a plan phase before start_after is changed to duration
	// and asserting in the next step that it is in fact deleted as per the migration
	breaks.add(stops{
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
	})
}
