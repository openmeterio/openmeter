package migrate_test

import (
	"database/sql"
	"encoding/json"
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
