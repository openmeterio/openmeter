package migrate_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/datetime"
	v20250624115812 "github.com/openmeterio/openmeter/tools/migrate/testdata/sqlcgen/20250624115812/db"
	v20250703081943 "github.com/openmeterio/openmeter/tools/migrate/testdata/sqlcgen/20250703081943/db"
)

func TestUsageResetAnchorTimesMigration(t *testing.T) {
	entId1 := ulid.Make()
	ent1UPAnchor := testutils.GetRFC3339Time(t, "2025-02-01T00:00:00Z")
	ent1ur1Id := ulid.Make()
	ent1ur2Id := ulid.Make()

	runner{
		stops: stops{
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
		},
	}.Test(t)
}

func TestEntitlementSubscriptionAnnotationMigration(t *testing.T) {
	entId := ulid.Make()
	featId := ulid.Make()
	subId := ulid.Make()
	custId := ulid.Make()
	phaseId := ulid.Make()
	itemId := ulid.Make()
	planId := ulid.Make()

	// Add IDs for the additional entitlements
	nonSubManagedEntId := ulid.Make()
	noItemEntId := ulid.Make()

	runner{
		stops: stops{
			{
				// before: 20250325115141_ent-subs-annotation.up.sql
				// Using the previous migration version
				version:   20250325114848,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// 1. Create a feature
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
							NOW(),
							NOW()
						)`,
						featId.String(),
					)
					require.NoError(t, err)

					// 2. Create a customer
					_, err = db.Exec(`
						INSERT INTO customers (
							namespace,
							id,
							created_at,
							updated_at,
							key,
							name
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'cust_1',
							'Customer 1'
						)`,
						custId.String(),
					)
					require.NoError(t, err)

					// 3. Create a plan
					_, err = db.Exec(`
						INSERT INTO plans (
							namespace,
							id,
							created_at,
							updated_at,
							key,
							version,
							name
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'plan_1',
							1,
							'Test Plan'
						)`,
						planId.String(),
					)
					require.NoError(t, err)

					// 4. Create a subscription
					_, err = db.Exec(`
						INSERT INTO subscriptions (
							namespace,
							id,
							created_at,
							updated_at,
							active_from,
							plan_id,
							currency,
							customer_id
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							NOW(),
							$2,
							'USD',
							$3
						)`,
						subId.String(),
						planId.String(),
						custId.String(),
					)
					require.NoError(t, err)

					// 5. Create a subscription phase
					_, err = db.Exec(`
						INSERT INTO subscription_phases (
							namespace,
							id,
							created_at,
							updated_at,
							key,
							name,
							active_from,
							subscription_id
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'phase_1',
							'Phase 1',
							NOW(),
							$2
						)`,
						phaseId.String(),
						subId.String(),
					)
					require.NoError(t, err)

					// 6. Create a subscription-managed entitlement without annotations
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
							subscription_managed
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'METERED',
							'feature_1',
							$2,
							'subject_1',
							TRUE
						)`,
						entId.String(),
						featId.String(),
					)
					require.NoError(t, err)

					// 7. Create a non-subscription-managed entitlement
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
							subscription_managed
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'METERED',
							'feature_1',
							$2,
							'subject_2',
							FALSE
						)`,
						nonSubManagedEntId.String(),
						featId.String(),
					)
					require.NoError(t, err)

					// 8. Create a subscription-managed entitlement without subscription item
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
							subscription_managed
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'METERED',
							'feature_1',
							$2,
							'subject_3',
							TRUE
						)`,
						noItemEntId.String(),
						featId.String(),
					)
					require.NoError(t, err)

					// 9. Create a subscription item linked to the entitlement
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
							entitlement_id
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							NOW(),
							'item_1',
							'Item 1',
							$2,
							$3
						)`,
						itemId.String(),
						phaseId.String(),
						entId.String(),
					)
					require.NoError(t, err)
				},
			},
			{
				// After our migration
				version:   20250325115141,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Check that the subscription.id annotation was added to the entitlement
					var annotationsJSON string
					err := db.QueryRow(`
						SELECT annotations::text
						FROM entitlements
						WHERE id = $1
					`, entId.String()).Scan(&annotationsJSON)
					require.NoError(t, err)

					// Parse the JSON and properly validate the structure
					annotations := make(map[string]interface{})
					err = json.Unmarshal([]byte(annotationsJSON), &annotations)
					require.NoError(t, err)

					// Verify the subscription.id is set correctly
					subscriptionID, ok := annotations["subscription.id"]
					require.True(t, ok, "subscription.id annotation not found")
					require.Equal(t, subId.String(), subscriptionID)

					// Check that non-subscription-managed entitlement doesn't have annotations
					var nonSubManagedAnnotations sql.NullString
					err = db.QueryRow(`
						SELECT annotations::text
						FROM entitlements
						WHERE id = $1
					`, nonSubManagedEntId.String()).Scan(&nonSubManagedAnnotations)
					require.NoError(t, err)
					require.False(t, nonSubManagedAnnotations.Valid, "non-subscription-managed entitlement should not have annotations")

					// Check that subscription-managed entitlement without item doesn't have annotations
					var noItemAnnotations sql.NullString
					err = db.QueryRow(`
						SELECT annotations::text
						FROM entitlements
						WHERE id = $1
					`, noItemEntId.String()).Scan(&noItemAnnotations)
					require.NoError(t, err)
					require.False(t, noItemAnnotations.Valid, "subscription-managed entitlement without item should not have annotations")
				},
			},
			{
				// After 20250325115141_ent-subs-annotation.down.sql
				version:   20250325114848,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					// Check that the subscription.id annotation was removed
					var annotationsJSON sql.NullString
					err := db.QueryRow(`
						SELECT annotations::text
						FROM entitlements
						WHERE id = $1
					`, entId.String()).Scan(&annotationsJSON)
					require.NoError(t, err)

					// If we have annotations (not NULL)
					if annotationsJSON.Valid {
						// Parse the JSON and verify subscription.id is not present
						annotations := make(map[string]interface{})
						err = json.Unmarshal([]byte(annotationsJSON.String), &annotations)
						require.NoError(t, err)

						_, hasSubscriptionId := annotations["subscription.id"]
						require.False(t, hasSubscriptionId, "subscription.id annotation should have been removed")
					}
					// If annotations is NULL, no need to check further as there's definitely no subscription.id
				},
			},
			{
				// We need to add back the subscription_managed=true to our entitlement as the column gets removed in later migration
				version:   20250325115141,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					// Update the subscription_managed value is set to true for all relevant entitlements
					_, err := db.Exec(`
						UPDATE entitlements
						SET subscription_managed = TRUE
						WHERE id IN ($1, $2)
					`, entId.String(), noItemEntId.String())
					require.NoError(t, err)
				},
			},
		},
	}.Test(t)
}

func TestBooleanEntitlementCountAnnotationMigration(t *testing.T) {
	// Create ULIDs for all entities we'll need
	featId := ulid.Make()
	entId := ulid.Make()
	subId := ulid.Make()
	custId := ulid.Make()
	phaseId := ulid.Make()
	itemId := ulid.Make()
	planId := ulid.Make()

	// Create a second subscription item with an existing annotations field
	itemId2 := ulid.Make()
	entId2 := ulid.Make()

	// Create a third subscription item that doesn't have a boolean entitlement
	itemId3 := ulid.Make()
	entId3 := ulid.Make()

	runner{
		stops: stops{
			{
				// Before our migration
				version:   20250422174622,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// 1. Create a feature
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
							NOW(),
							NOW()
						)`,
						featId.String(),
					)
					require.NoError(t, err)

					// 2. Create a customer
					_, err = db.Exec(`
						INSERT INTO customers (
							namespace,
							id,
							created_at,
							updated_at,
							key,
							name
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'cust_1',
							'Customer 1'
						)`,
						custId.String(),
					)
					require.NoError(t, err)

					// 3. Create a plan
					_, err = db.Exec(`
						INSERT INTO plans (
							namespace,
							id,
							created_at,
							updated_at,
							key,
							version,
							name
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'plan_1',
							1,
							'Test Plan'
						)`,
						planId.String(),
					)
					require.NoError(t, err)

					// 4. Create a subscription
					_, err = db.Exec(`
						INSERT INTO subscriptions (
							namespace,
							id,
							created_at,
							updated_at,
							active_from,
							plan_id,
							currency,
							customer_id
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							NOW(),
							$2,
							'USD',
							$3
						)`,
						subId.String(),
						planId.String(),
						custId.String(),
					)
					require.NoError(t, err)

					// 5. Create a subscription phase
					_, err = db.Exec(`
						INSERT INTO subscription_phases (
							namespace,
							id,
							created_at,
							updated_at,
							key,
							name,
							active_from,
							subscription_id
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'phase_1',
							'Phase 1',
							NOW(),
							$2
						)`,
						phaseId.String(),
						subId.String(),
					)
					require.NoError(t, err)

					// 6. Create boolean entitlements
					_, err = db.Exec(`
						INSERT INTO entitlements (
							namespace,
							id,
							created_at,
							updated_at,
							entitlement_type,
							feature_key,
							feature_id,
							subject_key
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'boolean',
							'feat_1',
							$2,
							'subject_1'
						)`,
						entId.String(),
						featId.String(),
					)
					require.NoError(t, err)

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
							annotations
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'boolean',
							'feat_1',
							$2,
							'subject_2',
							'{"existing.key": "some-value"}'
						)`,
						entId2.String(),
						featId.String(),
					)
					require.NoError(t, err)

					// Create a metered entitlement (not boolean)
					_, err = db.Exec(`
						INSERT INTO entitlements (
							namespace,
							id,
							created_at,
							updated_at,
							entitlement_type,
							feature_key,
							feature_id,
							subject_key
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							'metered',
							'feat_1',
							$2,
							'subject_3'
						)`,
						entId3.String(),
						featId.String(),
					)
					require.NoError(t, err)

					// 7. Create subscription items linked to the entitlements
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
							entitlement_id
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							NOW(),
							'item_1',
							'Item 1',
							$2,
							$3
						)`,
						itemId.String(),
						phaseId.String(),
						entId.String(),
					)
					require.NoError(t, err)

					// Second item with existing annotations
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
							entitlement_id,
							annotations
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							NOW(),
							'item_2',
							'Item 2',
							$2,
							$3,
							'{"existing.annotation": "value"}'
						)`,
						itemId2.String(),
						phaseId.String(),
						entId2.String(),
					)
					require.NoError(t, err)

					// Third item linked to a metered entitlement
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
							entitlement_id
						)
						VALUES (
							'default',
							$1,
							NOW(),
							NOW(),
							NOW(),
							'item_3',
							'Item 3',
							$2,
							$3
						)`,
						itemId3.String(),
						phaseId.String(),
						entId3.String(),
					)
					require.NoError(t, err)
				},
			},
			{
				// After our migration
				version:   20250424160933,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Check item with no previous annotations
					var annotationsJSON string
					err := db.QueryRow(`
						SELECT annotations::text
						FROM subscription_items
						WHERE id = $1
					`, itemId.String()).Scan(&annotationsJSON)
					require.NoError(t, err)

					annotations := make(map[string]interface{})
					err = json.Unmarshal([]byte(annotationsJSON), &annotations)
					require.NoError(t, err)

					// Verify the boolean count annotation is set correctly
					booleanCount, ok := annotations["subscription.entitlement.boolean.count"]
					require.True(t, ok, "subscription.entitlement.boolean.count annotation not found")
					require.Equal(t, float64(1), booleanCount) // JSON unmarshals numbers as float64

					// Check item with existing annotations
					err = db.QueryRow(`
						SELECT annotations::text
						FROM subscription_items
						WHERE id = $1
					`, itemId2.String()).Scan(&annotationsJSON)
					require.NoError(t, err)

					annotations = make(map[string]interface{})
					err = json.Unmarshal([]byte(annotationsJSON), &annotations)
					require.NoError(t, err)

					// Verify both annotations exist
					booleanCount, ok = annotations["subscription.entitlement.boolean.count"]
					require.True(t, ok, "subscription.entitlement.boolean.count annotation not found")
					require.Equal(t, float64(1), booleanCount)

					existingValue, ok := annotations["existing.annotation"]
					require.True(t, ok, "existing annotation not preserved")
					require.Equal(t, "value", existingValue)

					// Check that metered entitlement item doesn't have the boolean count annotation
					var nonBooleanAnnotations sql.NullString
					err = db.QueryRow(`
						SELECT annotations::text
						FROM subscription_items
						WHERE id = $1
					`, itemId3.String()).Scan(&nonBooleanAnnotations)
					require.NoError(t, err)

					if nonBooleanAnnotations.Valid {
						annotations = make(map[string]interface{})
						err = json.Unmarshal([]byte(nonBooleanAnnotations.String), &annotations)
						require.NoError(t, err)

						_, ok = annotations["subscription.entitlement.boolean.count"]
						require.False(t, ok, "non-boolean entitlement should not have boolean count annotation")
					}
				},
			},
			{
				// Test down migration
				version:   20250422174622,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					// Check that first item's annotation was removed
					var annotations sql.NullString
					err := db.QueryRow(`
						SELECT annotations::text
						FROM subscription_items
						WHERE id = $1
					`, itemId.String()).Scan(&annotations)
					require.NoError(t, err)

					// Since this item only had the boolean count annotation, it should now be NULL
					require.False(t, annotations.Valid, "annotations should be NULL after down migration")

					// Check that the second item's annotation was properly modified
					var annotationsJSON string
					err = db.QueryRow(`
						SELECT annotations::text
						FROM subscription_items
						WHERE id = $1
					`, itemId2.String()).Scan(&annotationsJSON)
					require.NoError(t, err)

					annotationsMap := make(map[string]interface{})
					err = json.Unmarshal([]byte(annotationsJSON), &annotationsMap)
					require.NoError(t, err)

					// Verify boolean count annotation was removed
					_, ok := annotationsMap["subscription.entitlement.boolean.count"]
					require.False(t, ok, "boolean count annotation should be removed")

					// Verify existing annotation still exists
					existingValue, ok := annotationsMap["existing.annotation"]
					require.True(t, ok, "existing annotation should be preserved")
					require.Equal(t, "value", existingValue)
				},
			},
		},
	}.Test(t)
}

func TestUsageResetUsagePeriodIntervalMigration(t *testing.T) {
	entId1 := ulid.Make()
	entId2 := ulid.Make()
	featId1 := ulid.Make()
	featId2 := ulid.Make()
	ur1Id := ulid.Make()
	ur2Id := ulid.Make()
	ur3Id := ulid.Make()

	runner{
		stops: stops{
			{
				// before: 20250703081943_entitlement-usageperiod-interval-change.up.sql
				version:   20250624115812, // Using the actual version before our migration
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Use SQLC-generated queries for type-safe database operations
					q := v20250624115812.New(db)
					ctx := context.Background()
					now := time.Now()

					// 1. Set up features
					err := q.CreateFeature(ctx, v20250624115812.CreateFeatureParams{
						Namespace: "default",
						ID:        featId1.String(),
						Key:       "feat_1",
						Name:      "Feature 1",
						CreatedAt: now,
						UpdatedAt: now,
					})
					require.NoError(t, err)

					err = q.CreateFeature(ctx, v20250624115812.CreateFeatureParams{
						Namespace: "default",
						ID:        featId2.String(),
						Key:       "feat_2",
						Name:      "Feature 2",
						CreatedAt: now,
						UpdatedAt: now,
					})
					require.NoError(t, err)

					// 2. Set up entitlements with different usage_period_intervals
					err = q.CreateEntitlement(ctx, v20250624115812.CreateEntitlementParams{
						Namespace:           "default",
						ID:                  entId1.String(),
						CreatedAt:           now,
						UpdatedAt:           now,
						EntitlementType:     "METERED",
						FeatureKey:          "feat_1",
						FeatureID:           featId1.String(),
						SubjectKey:          "subject_1",
						UsagePeriodInterval: sql.NullString{String: "P1W", Valid: true},
						UsagePeriodAnchor:   sql.NullTime{Time: now, Valid: true},
					})
					require.NoError(t, err)

					// Second entitlement with different interval
					err = q.CreateEntitlement(ctx, v20250624115812.CreateEntitlementParams{
						Namespace:           "default",
						ID:                  entId2.String(),
						CreatedAt:           now,
						UpdatedAt:           now,
						EntitlementType:     "METERED",
						FeatureKey:          "feat_2",
						FeatureID:           featId2.String(),
						SubjectKey:          "subject_2",
						UsagePeriodInterval: sql.NullString{String: "P1Y", Valid: true},
						UsagePeriodAnchor:   sql.NullTime{Time: now, Valid: true},
					})
					require.NoError(t, err)

					// 3. Set up usage resets
					err = q.CreateUsageReset(ctx, v20250624115812.CreateUsageResetParams{
						Namespace:     "default",
						ID:            ur1Id.String(),
						CreatedAt:     now,
						UpdatedAt:     now,
						EntitlementID: entId1.String(),
						ResetTime:     now,
						Anchor:        now,
					})
					require.NoError(t, err)

					err = q.CreateUsageReset(ctx, v20250624115812.CreateUsageResetParams{
						Namespace:     "default",
						ID:            ur2Id.String(),
						CreatedAt:     now,
						UpdatedAt:     now,
						EntitlementID: entId1.String(),
						ResetTime:     now,
						Anchor:        now,
					})
					require.NoError(t, err)

					err = q.CreateUsageReset(ctx, v20250624115812.CreateUsageResetParams{
						Namespace:     "default",
						ID:            ur3Id.String(),
						CreatedAt:     now,
						UpdatedAt:     now,
						EntitlementID: entId2.String(),
						ResetTime:     now,
						Anchor:        now,
					})
					require.NoError(t, err)

					// Verify that usage_period_interval column doesn't exist in usage_resets yet
					_, err = db.Exec(`SELECT usage_period_interval FROM usage_resets LIMIT 1`)
					require.Error(t, err, "usage_period_interval column should not exist before migration")
				},
			},
			{
				// After our migration
				version:   20250703081943,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					// Use SQLC-generated queries for the post-migration schema
					q := v20250703081943.New(db)
					ctx := context.Background()

					// Verify the column was added and populated correctly
					interval1, err := q.GetUsageResetInterval(ctx, ur1Id.String())
					require.NoError(t, err)
					require.Equal(t, "P1W", interval1, "usage_period_interval should match entitlement's interval")

					interval2, err := q.GetUsageResetInterval(ctx, ur2Id.String())
					require.NoError(t, err)
					require.Equal(t, "P1W", interval2, "usage_period_interval should match entitlement's interval")

					interval3, err := q.GetUsageResetInterval(ctx, ur3Id.String())
					require.NoError(t, err)
					require.Equal(t, "P1Y", interval3, "usage_period_interval should match entitlement's interval")

					// Verify the column is NOT NULL - try to insert with an empty string (which should be invalid)
					// Note: Since SQLC expects a string type, we can't actually pass NULL through it,
					// but we can test with raw SQL to verify the NOT NULL constraint
					_, err = db.Exec(`INSERT INTO usage_resets (namespace, id, created_at, updated_at, entitlement_id, reset_time, anchor, usage_period_interval) VALUES ('default', $1, NOW(), NOW(), $2, NOW(), NOW(), NULL)`, ulid.Make().String(), entId1.String())
					require.Error(t, err, "should not allow NULL values in usage_period_interval column")
				},
			},
			{
				// Test down migration
				version:   20250624115812,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					// Verify the column was dropped
					_, err := db.Exec(`SELECT usage_period_interval FROM usage_resets LIMIT 1`)
					require.Error(t, err, "usage_period_interval column should not exist after down migration")
				},
			},
		},
	}.Test(t)
}

func TestUsagePeriodIntervalDurationBackfillMigration(t *testing.T) {
	t.Run("om_func_generate_ulid", func(t *testing.T) {
		runner{
			stops: stops{
				{
					version:   20250723122351,
					direction: directionUp,
					action: func(t *testing.T, db *sql.DB) {
						// Let's fuzz it a bit
						for i := 0; i < 100; i++ {
							var ulid string

							err := db.QueryRow(`SELECT om_func_generate_ulid()`).Scan(&ulid)
							require.NoError(t, err)

							ulidRegex := regexp.MustCompile(`^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$`)
							require.True(t, ulidRegex.MatchString(ulid), "ULID should match regex")
						}
					},
				},
			},
		}.Test(t)
	})

	t.Run("om_func_go_add_date_normalized", func(t *testing.T) {
		runner{
			stops: stops{
				{
					version:   20250723122351,
					direction: directionUp,
					action: func(t *testing.T, db *sql.DB) {
						// Should add dates exactly as go's add_date
						tt := []struct {
							duration datetime.ISODurationString
							neg      bool
							date     time.Time
						}{
							// Adding only a small calendar duration
							{"P1D", false, time.Date(2025, 3, 12, 3, 0, 2, 0, time.UTC)},
							// Adding time components only
							{"PT2H3M", false, time.Date(2025, 3, 12, 3, 0, 2, 0, time.UTC)},
							// February is a 27 day month
							{"P1M", false, time.Date(2025, 1, 31, 3, 0, 2, 0, time.UTC)},
							{"P3M", false, time.Date(2025, 1, 31, 0, 8, 0, 0, time.UTC)},
							// June is a 30 day month
							{"P1M", false, time.Date(2025, 5, 31, 1, 0, 3, 0, time.UTC)},
							{"P3M", false, time.Date(2025, 5, 31, 0, 7, 0, 0, time.UTC)},
							// Just a random complex duration
							{"P1Y4M5DT2H3M6S", false, time.Date(2025, 1, 12, 3, 0, 2, 0, time.UTC)},
							// Add negative durations
							{"P1D", true, time.Date(2025, 3, 11, 3, 0, 2, 0, time.UTC)},
							{"P1W", true, time.Date(2025, 3, 11, 3, 0, 2, 0, time.UTC)},
							{"P1W", true, time.Date(2025, 3, 4, 3, 0, 2, 0, time.UTC)},
							{"P1M", true, time.Date(2025, 2, 28, 3, 0, 2, 0, time.UTC)},
							{"P1Y", true, time.Date(2024, 1, 12, 3, 0, 2, 0, time.UTC)},
							{"P1Y4M5DT2H3M6S", true, time.Date(2024, 8, 7, 0, 56, 54, 0, time.UTC)},
						}

						for _, tc := range tt {
							var res sql.NullTime
							query := `SELECT om_func_go_add_date_normalized($1,`
							if tc.neg {
								query += `$2::INTERVAL * -1`
							} else {
								query += `$2`
							}
							query += `);`

							require.NoError(t, db.QueryRow(query, tc.date, tc.duration).Scan(&res))
							require.True(t, res.Valid, "should return a valid time, got inputs: %v, %v", tc.date, tc.duration)

							duration, err := tc.duration.Parse()
							require.NoError(t, err)

							if tc.neg {
								duration = duration.Negate()
							}

							exp, _ := duration.Period.AddTo(tc.date)

							require.Equal(t, exp, res.Time.UTC(), "should add dates exactly as go's add_date, inputs: %v, %v", tc.date, tc.duration)
						}

						// PG does not support nanosecond resolution so we'll lose that
						{
							durStr := datetime.ISODurationString("P3M")
							at := time.Date(2025, 1, 31, 0, 8, 0, 1, time.UTC)

							var res sql.NullTime
							require.NoError(t, db.QueryRow(`SELECT om_func_go_add_date_normalized($1, $2)`, at, durStr).Scan(&res))
							require.True(t, res.Valid, "should return a valid time, got inputs: %v, %v", at, durStr)

							duration, err := durStr.Parse()
							require.NoError(t, err)

							exp, _ := duration.Period.AddTo(at)

							require.NotEqual(t, exp, res.Time.UTC())
							require.Equal(t, exp.Add(-time.Nanosecond), res.Time.UTC())
						}
					},
				},
			},
		}.Test(t)
	})

	t.Run("om_func_get_go_normalized_last_iteration_not_after_cutoff", func(t *testing.T) {
		runner{
			stops: stops{
				{
					version:   20250723122351,
					direction: directionUp,
					action: func(t *testing.T, db *sql.DB) {
						// Test if before cutoff
						{
							var res time.Time
							require.NoError(t, db.QueryRow(`SELECT om_func_get_go_normalized_last_iteration_not_after_cutoff($1, $2, $3)`,
								time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC),
								"P1M",
								time.Date(2025, 4, 12, 0, 0, 0, 0, time.UTC),
							).Scan(&res))

							require.Equal(t, time.Date(2025, 4, 5, 0, 0, 0, 0, time.UTC), res.UTC())
						}
						// Test if after cutoff
						{
							var res time.Time
							require.NoError(t, db.QueryRow(`SELECT om_func_get_go_normalized_last_iteration_not_after_cutoff($1, $2, $3)`,
								time.Date(2025, 7, 5, 0, 0, 0, 0, time.UTC),
								"P1M",
								time.Date(2025, 4, 12, 0, 0, 0, 0, time.UTC),
							).Scan(&res))

							require.Equal(t, time.Date(2025, 4, 5, 0, 0, 0, 0, time.UTC), res.UTC())
						}
						// Test if exactly on cutoff
						{
							var res time.Time
							require.NoError(t, db.QueryRow(`SELECT om_func_get_go_normalized_last_iteration_not_after_cutoff($1, $2, $3)`,
								time.Date(2025, 4, 12, 0, 0, 0, 0, time.UTC),
								"P1M",
								time.Date(2025, 4, 12, 0, 0, 0, 0, time.UTC),
							).Scan(&res))

							require.Equal(t, time.Date(2025, 4, 12, 0, 0, 0, 0, time.UTC), res.UTC())
						}
						// Test if will fall on cutoff
						{
							var res time.Time
							require.NoError(t, db.QueryRow(`SELECT om_func_get_go_normalized_last_iteration_not_after_cutoff($1, $2, $3)`,
								time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
								"P1M",
								time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC),
							).Scan(&res))

							// Notice the expected normalization
							require.Equal(t, time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC), res.UTC())
						}
						// Test the specific failing scenario: anchor after cutoff with P1W interval
						{
							var res time.Time
							require.NoError(t, db.QueryRow(`SELECT om_func_get_go_normalized_last_iteration_not_after_cutoff($1, $2, $3)`,
								testutils.GetRFC3339Time(t, "2024-11-06T19:30:00Z"), // anchor
								"P1W", // interval
								testutils.GetRFC3339Time(t, "2024-11-06T19:29:00Z"), // cutoff (1 minute before anchor)
							).Scan(&res))

							// Should return the last weekly iteration before the cutoff
							// Starting from 2024-11-06T19:30:00Z and going backwards by weeks
							// Should be 2024-10-30T19:30:00Z (one week before)
							require.Equal(t, testutils.GetRFC3339Time(t, "2024-10-30T19:30:00Z"), res.UTC())
						}
					},
				},
			},
		}.Test(t)
	})

	t.Run("om_func_update_usage_period_durations", func(t *testing.T) {
		now := time.Now()

		featId := ulid.Make()

		// Ent1 hasn't had any resets yet
		entId1 := ulid.Make()
		ent1MeasureUsageFrom := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
		// Ent2 had a single reset which is not aligned with the entitlement's anchor
		entId2 := ulid.Make()
		ent2MeasureUsageFrom := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
		ent2Ur1Id := ulid.Make()
		ent2Ur1ResetTime := time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC)
		// Ent3 has a single reset which is misaligned and reanchors
		entId3 := ulid.Make()
		ent3MeasureUsageFrom := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
		ent3Ur1Id := ulid.Make()
		ent3Ur1ResetTime := time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC)
		ent3Ur1Anchor := time.Date(2025, 3, 11, 0, 0, 0, 0, time.UTC)
		// Ent4 has a single reset which is misaligned, reanchors, and changes the interval
		entId4 := ulid.Make()
		ent4MeasureUsageFrom := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
		ent4Ur1Id := ulid.Make()
		ent4Ur1ResetTime := time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC)
		ent4Ur1Anchor := time.Date(2025, 3, 11, 0, 0, 0, 0, time.UTC)
		ent4Ur1Interval := "P1W" // This is not realistic but we need this never the less
		// Ent5 has two resets, changing to P1W then back to P1M
		// This is simply used as a more complex scenario to assert everything works as expected
		entId5 := ulid.Make()
		ent5MeasureUsageFrom := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
		ent5Ur1Id := ulid.Make()
		ent5Ur1ResetTime := time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC)
		ent5Ur1Anchor := time.Date(2025, 3, 11, 0, 0, 0, 0, time.UTC)
		ent5Ur1Interval := "P1W"
		ent5Ur2Id := ulid.Make()
		ent5Ur2ResetTime := time.Date(2025, 3, 23, 0, 0, 0, 0, time.UTC)
		ent5Ur2Anchor := time.Date(2025, 3, 23, 0, 0, 0, 0, time.UTC)
		ent5Ur2Interval := "P1M"
		// Ent6 has time components only in the interval
		// As the period is measure in hours, we'll set relative times close to current time
		entId6 := ulid.Make()
		ent6Interval := "PT3H"
		ent6MeasureUsageFrom := now.Truncate(time.Hour).Add(-time.Hour * 5)
		ent6Ur1Id := ulid.Make()
		ent6Ur1ResetTime := ent6MeasureUsageFrom.Add(time.Hour*2 + time.Minute*30)
		ent6Ur1Anchor := ent6MeasureUsageFrom
		ent6Ur1Interval := "PT1H"
		// Ent7 has both time and date components in the interval
		entId7 := ulid.Make()
		ent7Interval := "P1MT2H"
		ent7MeasureUsageFrom := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
		ent7Ur1Id := ulid.Make()
		ent7Ur1ResetTime := time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC)
		ent7Ur1Anchor := time.Date(2025, 3, 11, 0, 0, 0, 0, time.UTC)
		ent7Ur1Interval := "P1MT1H"

		runner{
			stops: stops{
				{
					// before version:   20250723122351,
					version:   20250703081943,
					direction: directionUp,
					// Let's do setup
					action: func(t *testing.T, db *sql.DB) {
						now := time.Now()

						q := v20250703081943.New(db)
						ctx := context.Background()

						// 1. Create a feature
						require.NoError(t, q.CreateFeature(
							ctx,
							v20250703081943.CreateFeatureParams{
								Namespace: "default",
								ID:        featId.String(),
								Key:       "feat_1",
								Name:      "Feature 1",
								CreatedAt: now,
								UpdatedAt: now,
							},
						))

						// 2. Create entitlements with different usage_period_intervals
						require.NoError(t, q.CreateEntitlement(
							ctx,
							v20250703081943.CreateEntitlementParams{
								Namespace:           "default",
								ID:                  entId1.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementType:     "metered",
								FeatureKey:          "feat_1",
								FeatureID:           featId.String(),
								SubjectKey:          "subject_1",
								UsagePeriodInterval: sql.NullString{String: "P1M", Valid: true},
								UsagePeriodAnchor:   sql.NullTime{Time: ent1MeasureUsageFrom, Valid: true},
								MeasureUsageFrom:    sql.NullTime{Time: ent1MeasureUsageFrom, Valid: true},
							},
						))

						require.NoError(t, q.CreateEntitlement(
							ctx,
							v20250703081943.CreateEntitlementParams{
								Namespace:           "default",
								ID:                  entId2.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementType:     "metered",
								FeatureKey:          "feat_1",
								FeatureID:           featId.String(),
								SubjectKey:          "subject_2",
								UsagePeriodInterval: sql.NullString{String: "P1M", Valid: true},
								UsagePeriodAnchor:   sql.NullTime{Time: ent2MeasureUsageFrom, Valid: true},
								MeasureUsageFrom:    sql.NullTime{Time: ent2MeasureUsageFrom, Valid: true},
							},
						))

						require.NoError(t, q.CreateEntitlement(
							ctx,
							v20250703081943.CreateEntitlementParams{
								Namespace:           "default",
								ID:                  entId3.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementType:     "metered",
								FeatureKey:          "feat_1",
								FeatureID:           featId.String(),
								SubjectKey:          "subject_3",
								UsagePeriodInterval: sql.NullString{String: "P1M", Valid: true},
								UsagePeriodAnchor:   sql.NullTime{Time: ent3MeasureUsageFrom, Valid: true},
								MeasureUsageFrom:    sql.NullTime{Time: ent3MeasureUsageFrom, Valid: true},
							},
						))

						require.NoError(t, q.CreateEntitlement(
							ctx,
							v20250703081943.CreateEntitlementParams{
								Namespace:           "default",
								ID:                  entId4.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementType:     "metered",
								FeatureKey:          "feat_1",
								FeatureID:           featId.String(),
								SubjectKey:          "subject_4",
								UsagePeriodInterval: sql.NullString{String: "P1M", Valid: true},
								UsagePeriodAnchor:   sql.NullTime{Time: ent4MeasureUsageFrom, Valid: true},
								MeasureUsageFrom:    sql.NullTime{Time: ent4MeasureUsageFrom, Valid: true},
							},
						))

						require.NoError(t, q.CreateEntitlement(
							ctx,
							v20250703081943.CreateEntitlementParams{
								Namespace:           "default",
								ID:                  entId5.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementType:     "metered",
								FeatureKey:          "feat_1",
								FeatureID:           featId.String(),
								SubjectKey:          "subject_5",
								UsagePeriodInterval: sql.NullString{String: "P1M", Valid: true},
								UsagePeriodAnchor:   sql.NullTime{Time: ent5MeasureUsageFrom, Valid: true},
								MeasureUsageFrom:    sql.NullTime{Time: ent5MeasureUsageFrom, Valid: true},
							},
						))

						require.NoError(t, q.CreateEntitlement(
							ctx,
							v20250703081943.CreateEntitlementParams{
								Namespace:           "default",
								ID:                  entId6.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementType:     "metered",
								FeatureKey:          "feat_1",
								FeatureID:           featId.String(),
								SubjectKey:          "subject_6",
								UsagePeriodInterval: sql.NullString{String: ent6Interval, Valid: true},
								UsagePeriodAnchor:   sql.NullTime{Time: ent6MeasureUsageFrom, Valid: true},
								MeasureUsageFrom:    sql.NullTime{Time: ent6MeasureUsageFrom, Valid: true},
							},
						))

						require.NoError(t, q.CreateEntitlement(
							ctx,
							v20250703081943.CreateEntitlementParams{
								Namespace:           "default",
								ID:                  entId7.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementType:     "metered",
								FeatureKey:          "feat_1",
								FeatureID:           featId.String(),
								SubjectKey:          "subject_7",
								UsagePeriodInterval: sql.NullString{String: ent7Interval, Valid: true},
								UsagePeriodAnchor:   sql.NullTime{Time: ent7MeasureUsageFrom, Valid: true},
								MeasureUsageFrom:    sql.NullTime{Time: ent7MeasureUsageFrom, Valid: true},
							},
						))

						// 3. Create usage resets
						// Ent 2
						require.NoError(t, q.CreateUsageResetWithInterval(
							ctx,
							v20250703081943.CreateUsageResetWithIntervalParams{
								Namespace:           "default",
								ID:                  ent2Ur1Id.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementID:       entId2.String(),
								Anchor:              ent2MeasureUsageFrom,
								ResetTime:           ent2Ur1ResetTime,
								UsagePeriodInterval: "P1M",
							},
						))

						// Ent 3
						require.NoError(t, q.CreateUsageResetWithInterval(
							ctx,
							v20250703081943.CreateUsageResetWithIntervalParams{
								Namespace:           "default",
								ID:                  ent3Ur1Id.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementID:       entId3.String(),
								Anchor:              ent3Ur1Anchor,
								ResetTime:           ent3Ur1ResetTime,
								UsagePeriodInterval: "P1M",
							},
						))

						// Ent 4
						require.NoError(t, q.CreateUsageResetWithInterval(
							ctx,
							v20250703081943.CreateUsageResetWithIntervalParams{
								Namespace:           "default",
								ID:                  ent4Ur1Id.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementID:       entId4.String(),
								Anchor:              ent4Ur1Anchor,
								ResetTime:           ent4Ur1ResetTime,
								UsagePeriodInterval: ent4Ur1Interval,
							},
						))

						// Ent 5
						require.NoError(t, q.CreateUsageResetWithInterval(
							ctx,
							v20250703081943.CreateUsageResetWithIntervalParams{
								Namespace:           "default",
								ID:                  ent5Ur1Id.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementID:       entId5.String(),
								Anchor:              ent5Ur1Anchor,
								ResetTime:           ent5Ur1ResetTime,
								UsagePeriodInterval: ent5Ur1Interval,
							},
						))

						require.NoError(t, q.CreateUsageResetWithInterval(
							ctx,
							v20250703081943.CreateUsageResetWithIntervalParams{
								Namespace:           "default",
								ID:                  ent5Ur2Id.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementID:       entId5.String(),
								Anchor:              ent5Ur2Anchor,
								ResetTime:           ent5Ur2ResetTime,
								UsagePeriodInterval: ent5Ur2Interval,
							},
						))

						// Ent 6
						require.NoError(t, q.CreateUsageResetWithInterval(
							ctx,
							v20250703081943.CreateUsageResetWithIntervalParams{
								Namespace:           "default",
								ID:                  ent6Ur1Id.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementID:       entId6.String(),
								Anchor:              ent6Ur1Anchor,
								ResetTime:           ent6Ur1ResetTime,
								UsagePeriodInterval: ent6Ur1Interval,
							},
						))

						// Ent 7
						require.NoError(t, q.CreateUsageResetWithInterval(
							ctx,
							v20250703081943.CreateUsageResetWithIntervalParams{
								Namespace:           "default",
								ID:                  ent7Ur1Id.String(),
								CreatedAt:           now,
								UpdatedAt:           now,
								EntitlementID:       entId7.String(),
								Anchor:              ent7Ur1Anchor,
								ResetTime:           ent7Ur1ResetTime,
								UsagePeriodInterval: ent7Ur1Interval,
							},
						))
					},
				},
				{
					version:   20250723122351,
					direction: directionUp,
					// Let's do assertions
					action: func(t *testing.T, db *sql.DB) {
						q := v20250703081943.New(db)
						ctx := context.Background()

						// NOTE: the now used here can be slightly off from the NOW() used in the migration
						// which can theoretically make the test flaky
						now := time.Now()

						// Entitlement 1 with no resets
						{
							ent1, err := q.GetEntitlementByID(ctx, entId1.String())
							require.NoError(t, err)

							usageResets, err := q.GetUsageResetsByEntitlementID(ctx, ent1.ID)
							require.NoError(t, err)

							// As the purpose of this migration is to fix the go date normalization behavior in the dataset
							// we'll use go date primitives to calculate periods

							// Let's start with how many resets there should be
							start := ent1MeasureUsageFrom
							monthlyIterCount := 0
							for iT := start; iT.Before(now); iT = iT.AddDate(0, 1, 0) {
								monthlyIterCount++
							}

							expectedResetCount := monthlyIterCount +
								0 // For the ones already present

							rstsJSON, err := json.MarshalIndent(usageResets, "", "  ")
							require.NoError(t, err)

							assert.Equal(t, expectedResetCount, len(usageResets), "Should have the correct number of usage resets, got %s", rstsJSON)

							// Let's check the first and second resets, for good measure
							firstReset := usageResets[0]
							secondReset := usageResets[1]

							// Let's assert that annotations were added
							for _, reset := range usageResets {
								assert.True(t, reset.Annotations.Valid, "Should have annotations, got %+v", reset)
								var ann map[string]string

								assert.NoError(t, json.Unmarshal(reset.Annotations.RawMessage, &ann))

								assert.Equal(t, "period_migration", ann["source"], "Should have the correct annotations, got %+v", reset)
							}

							// Let's assert the period info
							assert.Equal(t, time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC), firstReset.ResetTime.UTC(), "Should have the correct reset time, got %+v", firstReset)
							assert.Equal(t, "P31D", firstReset.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", firstReset)
							assert.Equal(t, time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC), firstReset.Anchor.UTC(), "Should have the correct anchor, got %+v", firstReset)

							assert.Equal(t, time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC), secondReset.ResetTime.UTC(), "Should have the correct reset time, got %+v", secondReset)
							assert.Equal(t, "P31D", secondReset.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", secondReset)
							assert.Equal(t, time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC), secondReset.Anchor.UTC(), "Should have the correct anchor, got %+v", secondReset)

							lastReset := usageResets[len(usageResets)-1]

							// The last reset should have the original interval string
							assert.Equal(t, "P1M", lastReset.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", lastReset)
							// The last reset should have the original anchor
							// This is so we're consistent with how billing handles the period change!
							assert.Equal(t, ent1MeasureUsageFrom, lastReset.Anchor.UTC(), "Should have the correct anchor, got %+v", lastReset)
						}

						// Entitlement 2 with a single reset (misaligned but NOT reanchoring)
						{
							ent2, err := q.GetEntitlementByID(ctx, entId2.String())
							require.NoError(t, err)

							usageResets, err := q.GetUsageResetsByEntitlementID(ctx, ent2.ID)
							require.NoError(t, err)

							// As the purpose of this migration is to fix the go date normalization behavior in the dataset
							// we'll use go date primitives to calculate periods

							// Let's start with how many resets there should be
							start := ent2MeasureUsageFrom

							// Note that this algo is the same as our single reset doesn't change the anchor
							monthlyIterCount := 0
							for iT := start; iT.Before(now); iT = iT.AddDate(0, 1, 0) {
								monthlyIterCount++
							}

							expectedResetCount := monthlyIterCount +
								1 // For the ones already present

							rstsJSON, err := json.MarshalIndent(usageResets, "", "  ")
							require.NoError(t, err)

							assert.Equal(t, expectedResetCount, len(usageResets), "Should have the correct number of usage resets, got %s", rstsJSON)

							// Let's check that our resets around the preexisting reset are correct
							justBefore := usageResets[1]
							updatedPreExisting := usageResets[2]
							justAfter := usageResets[3]

							// Normal, as with any other iteration
							assert.Equal(t, time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC), justBefore.ResetTime.UTC(), "Should have the correct reset time, got %+v", justBefore)
							assert.Equal(t, "P31D", justBefore.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", justBefore)

							// Let's assert it's the right one
							assert.Equal(t, time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC), updatedPreExisting.ResetTime.UTC(), "Should have the correct anchor, got %+v", updatedPreExisting)
							// The pre-existing has to be updated
							// in its ANCHOR TIME, which has to be normalized to the closest one-before iteration using
							// the old normalizing algo
							assert.Equal(t, time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC), updatedPreExisting.Anchor.UTC(), "Should have the correct anchor, got %+v", updatedPreExisting)
							assert.Equal(t, "P31D", updatedPreExisting.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", updatedPreExisting)

							// Normal, as with any other iteration, still using the same old anchor
							assert.Equal(t, time.Date(2025, 4, 3, 0, 0, 0, 0, time.UTC), justAfter.ResetTime.UTC(), "Should have the correct reset time, got %+v", justAfter)
							assert.Equal(t, "P30D", justAfter.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", justAfter)
						}

						// Entitlement 3 with a single reset (misaligned and reanchors)
						{
							ent3, err := q.GetEntitlementByID(ctx, entId3.String())
							require.NoError(t, err)

							usageResets, err := q.GetUsageResetsByEntitlementID(ctx, ent3.ID)
							require.NoError(t, err)

							// Let's just test that we behave correctly after the reanchoring reset
							updatedPreExisting := usageResets[2]
							justAfter := usageResets[3]

							// The pre-existing has to be updated
							assert.Equal(t, time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC), updatedPreExisting.ResetTime.UTC(), "Should have the correct reset time, got %+v", updatedPreExisting)
							assert.Equal(t, time.Date(2025, 3, 11, 0, 0, 0, 0, time.UTC), updatedPreExisting.Anchor.UTC(), "Should have the correct anchor, got %+v", updatedPreExisting)
							assert.Equal(t, "P31D", updatedPreExisting.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", updatedPreExisting)

							// The next one has to again be aligned with the new anchor
							assert.Equal(t, time.Date(2025, 4, 11, 0, 0, 0, 0, time.UTC), justAfter.ResetTime.UTC(), "Should have the correct reset time, got %+v", justAfter)
							assert.Equal(t, time.Date(2025, 4, 11, 0, 0, 0, 0, time.UTC), justAfter.Anchor.UTC(), "Should have the correct anchor, got %+v", justAfter)
							assert.Equal(t, "P30D", justAfter.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", justAfter)
						}

						// Entitlement 4 with a single reset (misaligned, reanchors and changes interval)
						{
							ent4, err := q.GetEntitlementByID(ctx, entId4.String())
							require.NoError(t, err)

							usageResets, err := q.GetUsageResetsByEntitlementID(ctx, ent4.ID)
							require.NoError(t, err)

							// Let's just test that we behave correctly after the reanchoring reset
							updatedPreExisting := usageResets[2]
							justAfter := usageResets[3]
							twoAfter := usageResets[4]

							// The pre-existing has to be updated
							assert.Equal(t, time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC), updatedPreExisting.ResetTime.UTC(), "Should have the correct reset time, got %+v", updatedPreExisting)
							assert.Equal(t, time.Date(2025, 3, 11, 0, 0, 0, 0, time.UTC), updatedPreExisting.Anchor.UTC(), "Should have the correct anchor, got %+v", updatedPreExisting)
							assert.Equal(t, "P7D", updatedPreExisting.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", updatedPreExisting)

							// The next one has to again be aligned with the new anchor
							assert.Equal(t, time.Date(2025, 3, 18, 0, 0, 0, 0, time.UTC), justAfter.ResetTime.UTC(), "Should have the correct reset time, got %+v", justAfter)
							assert.Equal(t, time.Date(2025, 3, 18, 0, 0, 0, 0, time.UTC), justAfter.Anchor.UTC(), "Should have the correct anchor, got %+v", justAfter)
							assert.Equal(t, "P7D", justAfter.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", justAfter)

							// The next one has to again be aligned with the new anchor
							assert.Equal(t, time.Date(2025, 3, 25, 0, 0, 0, 0, time.UTC), twoAfter.ResetTime.UTC(), "Should have the correct reset time, got %+v", twoAfter)
							assert.Equal(t, time.Date(2025, 3, 25, 0, 0, 0, 0, time.UTC), twoAfter.Anchor.UTC(), "Should have the correct anchor, got %+v", twoAfter)
							assert.Equal(t, "P7D", twoAfter.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", twoAfter)

							// The last one has to be the original interval
							lastReset := usageResets[len(usageResets)-1]
							// Note that due to PG reasons, P1W is translated to P7D but they are functionally identical
							assert.Equal(t, "P7D", lastReset.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", lastReset)
							assert.Equal(t, ent4Ur1Anchor, lastReset.Anchor.UTC(), "Should have the correct anchor, got %+v", lastReset)
						}

						// Entitlement 5, complex scenario
						{
							ent5, err := q.GetEntitlementByID(ctx, entId5.String())
							require.NoError(t, err)

							usageResets, err := q.GetUsageResetsByEntitlementID(ctx, ent5.ID)
							require.NoError(t, err)

							// Let's test for the first reset
							{
								// Let's just test that we behave correctly after the reanchoring reset
								updatedPreExisting := usageResets[2]
								justAfter := usageResets[3]

								// The pre-existing has to be updated
								assert.Equal(t, time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC), updatedPreExisting.ResetTime.UTC(), "Should have the correct reset time, got %+v", updatedPreExisting)
								assert.Equal(t, time.Date(2025, 3, 11, 0, 0, 0, 0, time.UTC), updatedPreExisting.Anchor.UTC(), "Should have the correct anchor, got %+v", updatedPreExisting)
								assert.Equal(t, "P7D", updatedPreExisting.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", updatedPreExisting)

								// The next one has to again be aligned with the new anchor
								assert.Equal(t, time.Date(2025, 3, 18, 0, 0, 0, 0, time.UTC), justAfter.ResetTime.UTC(), "Should have the correct reset time, got %+v", justAfter)
								assert.Equal(t, time.Date(2025, 3, 18, 0, 0, 0, 0, time.UTC), justAfter.Anchor.UTC(), "Should have the correct anchor, got %+v", justAfter)
								assert.Equal(t, "P7D", justAfter.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", justAfter)
							}

							// Let's test for the second reset
							{
								// Let's just test that we behave correctly after the reanchoring reset
								updatedPreExisting := usageResets[4]
								justAfter := usageResets[5]

								// The pre-existing has to be updated
								assert.Equal(t, time.Date(2025, 3, 23, 0, 0, 0, 0, time.UTC), updatedPreExisting.ResetTime.UTC(), "Should have the correct reset time, got %+v", updatedPreExisting)
								assert.Equal(t, time.Date(2025, 3, 23, 0, 0, 0, 0, time.UTC), updatedPreExisting.Anchor.UTC(), "Should have the correct anchor, got %+v", updatedPreExisting)
								assert.Equal(t, "P31D", updatedPreExisting.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", updatedPreExisting)

								// The next one has to again be aligned with the new anchor
								assert.Equal(t, time.Date(2025, 4, 23, 0, 0, 0, 0, time.UTC), justAfter.ResetTime.UTC(), "Should have the correct reset time, got %+v", justAfter)
								assert.Equal(t, time.Date(2025, 4, 23, 0, 0, 0, 0, time.UTC), justAfter.Anchor.UTC(), "Should have the correct anchor, got %+v", justAfter)
								assert.Equal(t, "P30D", justAfter.UsagePeriodInterval, "Should have the correct usage period interval, got %+v", justAfter)
							}
						}

						// Entitlement 6, time-only interval
						{
							ent6, err := q.GetEntitlementByID(ctx, entId6.String())
							require.NoError(t, err)

							usageResets, err := q.GetUsageResetsByEntitlementID(ctx, ent6.ID)
							require.NoError(t, err)

							ursJSON, err := json.MarshalIndent(usageResets, "", "  ")
							require.NoError(t, err)

							require.Len(t, usageResets, 5, "Should have the correct number of usage resets, got %s", ursJSON)

							nowMinus5HoursTruncated := now.Truncate(time.Hour).Add(-time.Hour * 5).UTC()

							// Let's make assertions for all expected items
							assert.Equal(t, nowMinus5HoursTruncated, usageResets[0].Anchor.UTC(), "Should have the correct anchor, got %+v", usageResets[0])
							assert.Equal(t, nowMinus5HoursTruncated, usageResets[0].ResetTime.UTC(), "Should have the correct reset time, got %+v", usageResets[0])

							assert.Equal(t, nowMinus5HoursTruncated.Add(time.Hour*2), usageResets[1].Anchor.UTC(), "Should have the correct anchor, got %+v", usageResets[1])
							assert.Equal(t, nowMinus5HoursTruncated.Add(time.Hour*2+time.Minute*30), usageResets[1].ResetTime.UTC(), "Should have the correct reset time, got %+v", usageResets[1])

							// Now it will realign itself to the anchor
							assert.Equal(t, nowMinus5HoursTruncated.Add(time.Hour*3), usageResets[2].Anchor.UTC(), "Should have the correct anchor, got %+v", usageResets[2])
							assert.Equal(t, nowMinus5HoursTruncated.Add(time.Hour*3), usageResets[2].ResetTime.UTC(), "Should have the correct reset time, got %+v", usageResets[2])

							assert.Equal(t, nowMinus5HoursTruncated.Add(time.Hour*4), usageResets[3].Anchor.UTC(), "Should have the correct anchor, got %+v", usageResets[3])
							assert.Equal(t, nowMinus5HoursTruncated.Add(time.Hour*4), usageResets[3].ResetTime.UTC(), "Should have the correct reset time, got %+v", usageResets[3])

							// And the last item will restore to the original anchor
							assert.Equal(t, nowMinus5HoursTruncated, usageResets[4].Anchor.UTC(), "Should have the correct anchor, got %+v", usageResets[4])
							assert.Equal(t, nowMinus5HoursTruncated.Add(time.Hour*5), usageResets[4].ResetTime.UTC(), "Should have the correct reset time, got %+v", usageResets[4])
						}

						// Entitlement 7, date+time interval
						{
							ent7, err := q.GetEntitlementByID(ctx, entId7.String())
							require.NoError(t, err)

							usageResets, err := q.GetUsageResetsByEntitlementID(ctx, ent7.ID)
							require.NoError(t, err)

							// Let's test we have the normalized interval
							assert.Equal(t, "P31DT2H", usageResets[0].UsagePeriodInterval, "Should have the correct usage period interval, got %+v", usageResets[0])
							assert.Equal(t, time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC), usageResets[0].Anchor.UTC(), "Should have the correct anchor, got %+v", usageResets[0])
							assert.Equal(t, time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC), usageResets[0].ResetTime.UTC(), "Should have the correct reset time, got %+v", usageResets[0])

							// Let's assert the usage reset update
							assert.Equal(t, time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC), usageResets[2].ResetTime.UTC(), "Should have the correct reset time, got %+v", usageResets[2])
							assert.Equal(t, time.Date(2025, 3, 11, 0, 0, 0, 0, time.UTC), usageResets[2].Anchor.UTC(), "Should have the correct anchor, got %+v", usageResets[2])
							assert.Equal(t, "P31DT1H", usageResets[2].UsagePeriodInterval, "Should have the correct usage period interval, got %+v", usageResets[2])

							// Let's assert it realigns correctly
							assert.Equal(t, time.Date(2025, 4, 11, 1, 0, 0, 0, time.UTC), usageResets[3].ResetTime.UTC(), "Should have the correct reset time, got %+v", usageResets[3])
							assert.Equal(t, time.Date(2025, 4, 11, 1, 0, 0, 0, time.UTC), usageResets[3].Anchor.UTC(), "Should have the correct anchor, got %+v", usageResets[3])
							assert.Equal(t, "P30DT1H", usageResets[3].UsagePeriodInterval, "Should have the correct usage period interval, got %+v", usageResets[3])
						}
					},
				},
			},
		}.Test(t)
	})

	t.Run("Should work on failing lines of dev dataset", func(t *testing.T) {
		runner{
			stops: stops{
				// Let's start with setup
				{
					// before version:   20250723122351,
					version:   20250703081943,
					direction: directionUp,
					// Let's do setup
					action: func(t *testing.T, db *sql.DB) {
						q := v20250703081943.New(db)
						ctx := context.Background()

						// 1. Create (some) Feature so it will work
						require.NoError(t, q.CreateFeature(
							ctx,
							v20250703081943.CreateFeatureParams{
								Namespace: "org_2l3uuzkgTdvCyom82y11jeZO2u5",
								ID:        "01J5ZSQF319B1M61GNH9ZBG23D",
								Key:       "total_api_usage",
								Name:      "total_api_usage",
								CreatedAt: testutils.GetRFC3339Time(t, "2024-11-06T19:29:00Z"),
								UpdatedAt: testutils.GetRFC3339Time(t, "2024-11-06T19:29:00Z"),
							},
						))

						// 2. Create the entitlement
						require.NoError(t, q.CreateEntitlement(
							ctx,
							v20250703081943.CreateEntitlementParams{
								ID:              "01JC1F7J8FXTX0YGVNB5Y4CH11",
								Namespace:       "org_2l3uuzkgTdvCyom82y11jeZO2u5",
								CreatedAt:       testutils.GetRFC3339Time(t, "2024-11-06T19:29:11Z"),
								UpdatedAt:       testutils.GetRFC3339Time(t, "2024-11-06T19:32:04Z"),
								FeatureID:       "01J5ZSQF319B1M61GNH9ZBG23D",
								SubjectKey:      "lol",
								EntitlementType: "metered",
								FeatureKey:      "total_api_usage",
								MeasureUsageFrom: sql.NullTime{
									Time:  testutils.GetRFC3339Time(t, "2024-11-06T19:29:00Z"),
									Valid: true,
								},
								UsagePeriodInterval: sql.NullString{
									String: "P1W",
									Valid:  true,
								},
								UsagePeriodAnchor: sql.NullTime{
									Time:  testutils.GetRFC3339Time(t, "2024-11-06T19:30:00Z"),
									Valid: true,
								},
							},
						))

						// 3. Let's create the usage resets
						require.NoError(t, q.CreateUsageResetWithInterval(
							ctx,
							v20250703081943.CreateUsageResetWithIntervalParams{
								ID:                  "01JC1FAW6XTAGF9BDRSMC79VQC",
								Namespace:           "org_2l3uuzkgTdvCyom82y11jeZO2u5",
								CreatedAt:           testutils.GetRFC3339Time(t, "2024-11-06T19:30:59Z"),
								UpdatedAt:           testutils.GetRFC3339Time(t, "2024-11-06T19:30:59Z"),
								ResetTime:           testutils.GetRFC3339Time(t, "2024-11-06T19:30:00Z"),
								EntitlementID:       "01JC1F7J8FXTX0YGVNB5Y4CH11",
								Anchor:              testutils.GetRFC3339Time(t, "2024-11-06T19:30:00Z"),
								UsagePeriodInterval: "P1W",
							},
						))
					},
				},
				// And now do assertions
				// {
				// 	version:   20250723122351,
				// 	direction: directionUp,
				// 	// Let's do assertions
				// 	action: func(t *testing.T, db *sql.DB) {
				// 	},
				// },
			},
		}.Test(t)
	})
}
