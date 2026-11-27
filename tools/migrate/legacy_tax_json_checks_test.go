package migrate_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const (
	dropLegacyTaxJSONChecksPreviousVersion = 20261125224333
	dropLegacyTaxJSONChecksVersion         = 20261127120811
)

func TestDropLegacyTaxJSONChecksMigration(t *testing.T) {
	tests := []struct {
		name        string
		table       string
		constraints []string
		// seedParent creates the parent the row hangs off (phase or add-on) and returns its ID.
		seedParent func(t *testing.T, db *sql.DB, namespace string) string
		insert     func(t *testing.T, db *sql.DB, namespace, parentID, id string, taxCodeID, taxBehavior, taxJSON any) (sql.Result, error)
	}{
		{
			name:        "plan rate card",
			table:       "plan_rate_cards",
			constraints: []string{"plan_rate_card_tax_code_consistency", "plan_rate_card_tax_behavior_consistency"},
			seedParent: func(t *testing.T, db *sql.DB, namespace string) string {
				phaseID := ulid.Make().String()
				seedPlan(t, db, namespace, phaseID)

				return phaseID
			},
			insert: func(t *testing.T, db *sql.DB, namespace, parentID, id string, taxCodeID, taxBehavior, taxJSON any) (sql.Result, error) {
				return insertPlanRateCard(t, db, namespace, parentID, id, id, taxCodeID, taxBehavior, taxJSON)
			},
		},
		{
			name:        "add-on rate card",
			table:       "addon_rate_cards",
			constraints: []string{"addon_rate_card_tax_code_consistency", "addon_rate_card_tax_behavior_consistency"},
			seedParent: func(t *testing.T, db *sql.DB, namespace string) string {
				addonID := ulid.Make().String()
				seedAddon(t, db, namespace, addonID)

				return addonID
			},
			insert: func(t *testing.T, db *sql.DB, namespace, parentID, id string, taxCodeID, taxBehavior, taxJSON any) (sql.Result, error) {
				return insertAddonRateCard(t, db, namespace, parentID, id, id, taxCodeID, taxBehavior, taxJSON)
			},
		},
		{
			name:        "subscription item",
			table:       "subscription_items",
			constraints: []string{"subscription_item_tax_code_consistency", "subscription_item_tax_behavior_consistency"},
			seedParent: func(t *testing.T, db *sql.DB, namespace string) string {
				phaseID := ulid.Make().String()
				seedSubscription(t, db, namespace, phaseID)

				return phaseID
			},
			insert: func(t *testing.T, db *sql.DB, namespace, parentID, id string, taxCodeID, taxBehavior, taxJSON any) (sql.Result, error) {
				return insertSubscriptionItem(t, db, namespace, parentID, id, id, taxCodeID, taxBehavior, taxJSON)
			},
		},
		{
			name:        "billing workflow config",
			table:       "billing_workflow_configs",
			constraints: []string{"billing_workflow_config_tax_code_consistency", "billing_workflow_config_tax_behavior_consistency"},
			seedParent: func(*testing.T, *sql.DB, string) string {
				return ""
			},
			insert: func(t *testing.T, db *sql.DB, namespace, _, id string, taxCodeID, taxBehavior, taxJSON any) (sql.Result, error) {
				return insertBillingWorkflowConfig(t, db, namespace, id, taxCodeID, taxBehavior, taxJSON)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, migrator := newMigrationTestEnv(t)

			// given: two tax codes and a row whose JSON agrees with its columns, as the old checks required
			require.NoError(t, migrator.Migrate(dropLegacyTaxJSONChecksPreviousVersion))

			namespace := "drop_legacy_tax_json_checks"
			createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			firstTaxCodeID := ulid.Make().String()
			secondTaxCodeID := ulid.Make().String()
			seedTaxCode(t, db, namespace, firstTaxCodeID, "first", "First", `[]`, nil, createdAt, sql.NullTime{})
			seedTaxCode(t, db, namespace, secondTaxCodeID, "second", "Second", `[]`, nil, createdAt, sql.NullTime{})

			parentID := tt.seedParent(t, db, namespace)
			agreeingID := ulid.Make().String()
			_, err := tt.insert(t, db, namespace, parentID, agreeingID, firstTaxCodeID, "inclusive",
				`{"tax_code_id": "`+firstTaxCodeID+`", "behavior": "inclusive"}`)
			require.NoError(t, err)

			// when: migrating up
			require.NoError(t, migrator.Migrate(dropLegacyTaxJSONChecksVersion))

			// then: a row with the columns set and no JSON is accepted
			_, err = tt.insert(t, db, namespace, parentID, ulid.Make().String(), firstTaxCodeID, "exclusive", nil)
			require.NoError(t, err)

			// then: changing the columns without touching the old JSON is accepted
			_, err = db.ExecContext(t.Context(),
				`UPDATE `+tt.table+` SET tax_code_id = $1, tax_behavior = 'exclusive' WHERE id = $2`,
				secondTaxCodeID, agreeingID)
			require.NoError(t, err)

			// then: the constraints are gone
			require.Empty(t, presentConstraints(t, db, tt.table, tt.constraints))

			// when: migrating back down
			require.NoError(t, migrator.Migrate(dropLegacyTaxJSONChecksPreviousVersion))

			// then: the constraints are back but unvalidated, so the rows above, which break them, survive
			for name, validated := range presentConstraints(t, db, tt.table, tt.constraints) {
				require.False(t, validated, name)
			}

			require.Len(t, presentConstraints(t, db, tt.table, tt.constraints), len(tt.constraints))
		})
	}
}

// presentConstraints returns which of the named constraints exist on the table, mapped to whether
// they are validated (false means NOT VALID).
func presentConstraints(t *testing.T, db *sql.DB, table string, names []string) map[string]bool {
	t.Helper()

	present := map[string]bool{}
	for _, name := range names {
		var validated bool
		err := db.QueryRowContext(t.Context(), `
			SELECT convalidated FROM pg_constraint
			WHERE conrelid = $1::regclass AND conname = $2
		`, table, name).Scan(&validated)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}

		require.NoError(t, err)

		present[name] = validated
	}

	return present
}
