package migrate_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// RateCardDetails holds the details of a rate card for testing
type RateCardDetails struct {
	ID                          string
	Namespace                   string
	Metadata                    []byte
	Name                        string
	Description                 sql.NullString
	Key                         string
	EntitlementType             string
	EntitlementTemplateMetadata []byte
	IsSoftLimit                 sql.NullBool
	IssueAfterReset             sql.NullFloat64
	UsagePeriod                 sql.NullString
	Type                        string
	FeatureKey                  string
	FeatureID                   string
	TaxConfig                   []byte
	BillingCadence              sql.NullString
	Price                       []byte
	Discounts                   []byte
	OriginalID                  sql.NullString
}

// fetchAndVerifyRateCard fetches a rate card by key and table and verifies its properties
func fetchAndVerifyRateCard(t *testing.T, db *sql.DB, key, tableName, originalIDField string, expectedValues map[string]interface{}) RateCardDetails {
	t.Helper()

	var tableJoin, whereClause string
	if tableName != "" {
		tableAlias := tableName[0:1] // First letter of table name as alias
		tableJoin = fmt.Sprintf("JOIN %s %s ON r.id = %s.ratecard_id", tableName, tableAlias, tableAlias)
	}

	var originalIDSelect string
	if originalIDField != "" {
		originalIDSelect = fmt.Sprintf("r.metadata->>'%s' as original_id", originalIDField)
	} else {
		originalIDSelect = "NULL::text as original_id"
	}

	whereClause = fmt.Sprintf("WHERE r.key = '%s'", key)

	var rateCard RateCardDetails
	query := fmt.Sprintf(`
		SELECT
			r.id,
			r.namespace,
			r.metadata,
			r.name,
			r.description,
			r.key,
			r.entitlement_template_entitlement_type,
			r.entitlement_template_metadata,
			r.entitlement_template_is_soft_limit,
			r.entitlement_template_issue_after_reset,
			r.entitlement_template_usage_period,
			r.type,
			r.feature_key,
			r.feature_id,
			r.tax_config,
			r.billing_cadence,
			r.price,
			r.discounts,
			%s
		FROM rate_cards r
		%s
		%s
	`, originalIDSelect, tableJoin, whereClause)

	err := db.QueryRow(query).Scan(
		&rateCard.ID,
		&rateCard.Namespace,
		&rateCard.Metadata,
		&rateCard.Name,
		&rateCard.Description,
		&rateCard.Key,
		&rateCard.EntitlementType,
		&rateCard.EntitlementTemplateMetadata,
		&rateCard.IsSoftLimit,
		&rateCard.IssueAfterReset,
		&rateCard.UsagePeriod,
		&rateCard.Type,
		&rateCard.FeatureKey,
		&rateCard.FeatureID,
		&rateCard.TaxConfig,
		&rateCard.BillingCadence,
		&rateCard.Price,
		&rateCard.Discounts,
		&rateCard.OriginalID,
	)
	require.NoError(t, err)

	// Verify expected values
	for field, expectedValue := range expectedValues {
		switch field {
		case "Namespace":
			require.Equal(t, expectedValue, rateCard.Namespace, "Namespace should match")
		case "Name":
			require.Equal(t, expectedValue, rateCard.Name, "Name should match")
		case "Description":
			require.Equal(t, expectedValue, rateCard.Description.String, "Description should match")
			require.True(t, rateCard.Description.Valid, "Description should be valid")
		case "Key":
			require.Equal(t, expectedValue, rateCard.Key, "Key should match")
		case "EntitlementType":
			require.Equal(t, expectedValue, rateCard.EntitlementType, "EntitlementType should match")
		case "Type":
			require.Equal(t, expectedValue, rateCard.Type, "Type should match")
		case "FeatureKey":
			require.Equal(t, expectedValue, rateCard.FeatureKey, "FeatureKey should match")
		case "BillingCadence":
			require.Equal(t, expectedValue, rateCard.BillingCadence.String, "BillingCadence should match")
			require.True(t, rateCard.BillingCadence.Valid, "BillingCadence should be valid")
		case "UsagePeriod":
			require.Equal(t, expectedValue, rateCard.UsagePeriod.String, "UsagePeriod should match")
			require.True(t, rateCard.UsagePeriod.Valid, "UsagePeriod should be valid")
		case "IssueAfterReset":
			require.Equal(t, expectedValue, rateCard.IssueAfterReset.Float64, "IssueAfterReset should match")
			require.True(t, rateCard.IssueAfterReset.Valid, "IssueAfterReset should be valid")
		case "IsSoftLimit":
			require.Equal(t, expectedValue, rateCard.IsSoftLimit.Bool, "IsSoftLimit should match")
			require.True(t, rateCard.IsSoftLimit.Valid, "IsSoftLimit should be valid")
		case "HasOriginalID":
			if expectedValue.(bool) {
				require.NotEmpty(t, rateCard.OriginalID.String, "OriginalID should not be empty in metadata")
				require.True(t, rateCard.OriginalID.Valid, "OriginalID should be valid")
			} else {
				require.False(t, rateCard.OriginalID.Valid, "OriginalID should be NULL")
			}
		case "MetadataContains":
			for key, val := range expectedValue.(map[string]string) {
				if key == "EntitlementTemplateMetadata" {
					require.Contains(t, string(rateCard.EntitlementTemplateMetadata), val, "EntitlementTemplateMetadata should contain expected value")
				} else if key == "Metadata" {
					require.Contains(t, string(rateCard.Metadata), val, "Metadata should contain expected value")
				}
			}
		}
	}

	// Generic assertions that apply to all rate cards
	require.NotEmpty(t, rateCard.ID, "ID should not be empty")
	require.NotEmpty(t, rateCard.FeatureID, "FeatureID should not be empty")

	// Check JSON fields if they should be present
	if rateCard.TaxConfig != nil {
		require.NotEmpty(t, rateCard.TaxConfig, "TaxConfig should not be empty")
	}
	if rateCard.Price != nil {
		require.NotEmpty(t, rateCard.Price, "Price should not be empty")
	}

	return rateCard
}

func TestRateCardMigrationDataSetup(t *testing.T) {
	runner{stops{
		{
			// Create test data before our migration
			version:   20250506140848, // This is the migration that adds ratecard_id to the table
			direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				t.Log("Setting up test data for RateCard migration")

				// First create a plan for the phases
				planID := ulid.Make().String()
				_, err := db.Exec(`
					INSERT INTO plans (
						id, namespace, metadata, created_at, updated_at,
						name, key, version, currency
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'Test Plan', 'test-plan', 1, 'USD'
					)`, planID)
				require.NoError(t, err)

				// Create test plan phase for plan rate cards
				phaseID := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO plan_phases (
						id, namespace, metadata, created_at, updated_at,
						name, key, plan_id, index, duration
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'Test Phase', 'test-phase', $2, 0, 'P1M'
					)`, phaseID, planID)
				require.NoError(t, err)

				// Create features for the rate cards
				featureID := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO features (
						id, namespace, metadata, created_at, updated_at,
						name, key
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'Test Feature', 'test-feature'
					)`, featureID)
				require.NoError(t, err)

				featureAPICALLID := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO features (
						id, namespace, metadata, created_at, updated_at,
						name, key
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'API Calls', 'api-calls'
					)`, featureAPICALLID)
				require.NoError(t, err)

				featureSeatsID := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO features (
						id, namespace, metadata, created_at, updated_at,
						name, key
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'Seats', 'seats'
					)`, featureSeatsID)
				require.NoError(t, err)

				// Create test addon for addon rate cards
				addonID := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO addons (
						id, namespace, metadata, created_at, updated_at,
						name, key, currency, instance_type, version
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'Test Addon', 'test-addon', 'USD', 'single', 1
					)`, addonID)
				require.NoError(t, err)

				// Create a customer for subscription
				customerID := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO customers (
						id, namespace, metadata, created_at, updated_at,
						name, key, currency
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'Test Customer', 'test-customer', 'USD'
					)`, customerID)
				require.NoError(t, err)

				// Create a subscription
				subscriptionID := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO subscriptions (
						id, namespace, metadata, created_at, updated_at,
						name, active_from, currency, customer_id, plan_id
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'Test Subscription', NOW(), 'USD', $2, $3
					)`, subscriptionID, customerID, planID)
				require.NoError(t, err)

				// Create a subscription phase
				subscriptionPhaseID := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO subscription_phases (
						id, namespace, metadata, created_at, updated_at,
						key, name, active_from, subscription_id
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'test-phase', 'Test Phase', NOW(), $2
					)`, subscriptionPhaseID, subscriptionID)
				require.NoError(t, err)

				// 1. Plan RateCards - Usage based type with metered entitlement template
				planRateCardID1 := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO plan_rate_cards (
						id, namespace, metadata, created_at, updated_at,
						name, description, key, type, feature_key, feature_id,
						entitlement_template, tax_config, billing_cadence, price, discounts,
						phase_id
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'Plan RateCard 1', 'Usage Based with Metered', 'plan-ratecard-1', 'usage_based', 'api-calls', $3,
						'{"entitlementType":"metered","issueAfterReset":1000,"usagePeriod":"P1M", "metadata":{"test":"test"}}',
						'{"stripe":{"code":"txcd_99999999"}}', 'P1M',
						'{"type":"unit","amount":"10.00"}', '{"percentage":{"percentage":25}}',
						$2
					)`,
					planRateCardID1, phaseID, featureAPICALLID,
				)
				require.NoError(t, err)

				// 2. Plan RateCards - Flat fee type with static entitlement template
				planRateCardID2 := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO plan_rate_cards (
						id, namespace, metadata, created_at, updated_at,
						name, description, key, type, feature_key, feature_id,
						entitlement_template, tax_config, billing_cadence, price, discounts,
						phase_id
					) VALUES (
						$2, 'default', '{}', NOW(), NOW(),
						'Plan RateCard 2', 'Flat Fee with Static', 'plan-ratecard-2', 'flat_fee', 'seats', $3,
						'{"entitlementType":"static","config":"{\"limit\":100}"}',
						'{"stripe":{"code":"txcd_88888888"}}', 'P1M',
						'{"type":"flat","amount":"99.00","paymentTerm":"in_advance"}', NULL,
						$1
					)`,
					phaseID, planRateCardID2, featureSeatsID,
				)
				require.NoError(t, err)

				// 3. Addon RateCards - Usage based with boolean entitlement
				addonRateCardID1 := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO addon_rate_cards (
						id, namespace, metadata, created_at, updated_at,
						name, description, key, type, feature_key, feature_id,
						entitlement_template, tax_config, billing_cadence, price, discounts,
						addon_id
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'Addon RateCard 1', 'Usage Based with Boolean', 'addon-ratecard-1', 'usage_based', 'storage', $3,
						'{"entitlementType":"boolean","config":"{\"enabled\":true}"}',
						'{"stripe":{"code":"txcd_77777777"}}', 'P1M',
						'{"type":"tiered","tiers":[{"upTo":10,"unitAmount":"5.00"},{"upTo":null,"unitAmount":"3.00"}]}', NULL,
						$2
					)`,
					addonRateCardID1, addonID, featureID,
				)
				require.NoError(t, err)

				// 4. Addon RateCards - Flat fee with metadata
				addonRateCardID2 := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO addon_rate_cards (
						id, namespace, metadata, created_at, updated_at,
						name, description, key, type, feature_key, feature_id,
						entitlement_template, tax_config, billing_cadence, price, discounts,
						addon_id
					) VALUES (
						$1, 'enterprise', '{"premium":true}', NOW(), NOW(),
						'Addon RateCard 2', 'Flat Fee with Metadata', 'addon-ratecard-2', 'flat_fee', 'premium-support', $3,
						'{"entitlementType":"static","metadata":{"priority":"high"},"config":"{\"level\":2}"}',
						'{"stripe":{"code":"txcd_66666666"}}', 'P1Y',
						'{"type":"flat","amount":"499.00","paymentTerm":"in_arrears"}', '{"usage":{"quantity":"10"}}',
						$2
					)`,
					addonRateCardID2, addonID, featureID,
				)
				require.NoError(t, err)

				// 5. Subscription Items - With various fields set
				subscriptionItemID1 := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO subscription_items (
						id, namespace, created_at, updated_at,
						active_from, key, name, description, feature_key,
						entitlement_template, tax_config, billing_cadence, price, discounts,
						phase_id
					) VALUES (
						$1, 'default', NOW(), NOW(),
						NOW(), 'sub-item-1', 'Sub Item 1', 'With all fields', 'seats',
						'{"entitlementType":"static","isSoftLimit":true,"config":"{\"value\":5}"}',
						'{"stripe":{"code":"txcd_55555555"}}', 'P1M',
						'{"type":"flat","amount":"50.00","paymentTerm":"in_arrears"}', '{"percentage":{"percentage":10}}',
						$2
					)`,
					subscriptionItemID1, subscriptionPhaseID,
				)
				require.NoError(t, err)

				// 6. Subscription Items - With minimal fields
				subscriptionItemID2 := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO subscription_items (
						id, namespace, created_at, updated_at,
						active_from, key, name, feature_key,
						entitlement_template, phase_id
					) VALUES (
						$1, 'default', NOW(), NOW(),
						NOW(), 'sub-item-2', 'Sub Item 2', 'api-calls',
						'{"entitlementType":"metered","usagePeriod":"P1M"}',
						$2
					)`,
					subscriptionItemID2, subscriptionPhaseID,
				)
				require.NoError(t, err)

				// 7. Create one plan rate card that already has a reference to a rate_card
				// This tests that our migration doesn't disturb existing references
				existingRateCardID := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO rate_cards (
						id, namespace, metadata, created_at, updated_at,
						name, key, entitlement_template_entitlement_type,
						type, feature_key, feature_id
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'Existing Rate Card', 'existing-rate-card', 'metered',
						'usage_based', 'existing-feature', $2
					)`,
					existingRateCardID, featureID,
				)
				require.NoError(t, err)

				planRateCardWithRefID := ulid.Make().String()
				_, err = db.Exec(`
					INSERT INTO plan_rate_cards (
						id, namespace, metadata, created_at, updated_at,
						name, key, type, feature_key,
						entitlement_template, phase_id, ratecard_id
					) VALUES (
						$1, 'default', '{}', NOW(), NOW(),
						'Plan With Ref', 'plan-with-ref', 'usage_based', 'existing-feature',
						'{"entitlementType":"metered"}', $2, $3
					)`,
					planRateCardWithRefID, phaseID, existingRateCardID,
				)
				require.NoError(t, err)

				// Count our test data to verify setup
				var count int
				err = db.QueryRow(`SELECT COUNT(*) FROM plan_rate_cards`).Scan(&count)
				require.NoError(t, err)
				require.Equal(t, 3, count, "Should have 3 plan rate cards")

				err = db.QueryRow(`SELECT COUNT(*) FROM addon_rate_cards`).Scan(&count)
				require.NoError(t, err)
				require.Equal(t, 2, count, "Should have 2 addon rate cards")

				err = db.QueryRow(`SELECT COUNT(*) FROM subscription_items`).Scan(&count)
				require.NoError(t, err)
				require.Equal(t, 2, count, "Should have 2 subscription items")

				err = db.QueryRow(`SELECT COUNT(*) FROM rate_cards`).Scan(&count)
				require.NoError(t, err)
				require.Equal(t, 1, count, "Should have 1 existing rate card")

				t.Log("Successfully set up test data for RateCard migration")
			},
		},
		{
			// Verify the migration worked correctly
			version:   20250506141154, // The migration we're testing
			direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				t.Log("Verifying rate cards migration results")

				// Count the number of rate cards - should now be 7 (1 existing + 2 plan + 2 addon + 2 subscription)
				var rateCardCount int
				err := db.QueryRow(`SELECT COUNT(*) FROM rate_cards`).Scan(&rateCardCount)
				require.NoError(t, err)
				require.Equal(t, 7, rateCardCount, "Should have 7 rate cards after migration")

				// Verify that all plan_rate_cards now have ratecard_id values
				var planRateCardsWithRefs int
				err = db.QueryRow(`SELECT COUNT(*) FROM plan_rate_cards WHERE ratecard_id IS NOT NULL`).Scan(&planRateCardsWithRefs)
				require.NoError(t, err)
				require.Equal(t, 3, planRateCardsWithRefs, "All plan_rate_cards should have ratecard_id references")

				// Verify that all addon_rate_cards now have ratecard_id values
				var addonRateCardsWithRefs int
				err = db.QueryRow(`SELECT COUNT(*) FROM addon_rate_cards WHERE ratecard_id IS NOT NULL`).Scan(&addonRateCardsWithRefs)
				require.NoError(t, err)
				require.Equal(t, 2, addonRateCardsWithRefs, "All addon_rate_cards should have ratecard_id references")

				// Verify that all subscription_items now have ratecard_id values
				var subscriptionItemsWithRefs int
				err = db.QueryRow(`SELECT COUNT(*) FROM subscription_items WHERE ratecard_id IS NOT NULL`).Scan(&subscriptionItemsWithRefs)
				require.NoError(t, err)
				require.Equal(t, 2, subscriptionItemsWithRefs, "All subscription_items should have ratecard_id references")

				// Verify that rate_cards have the correct entitlement_template_entitlement_type values
				var metered, static, boolean int
				err = db.QueryRow(`SELECT COUNT(*) FROM rate_cards WHERE entitlement_template_entitlement_type = 'metered'`).Scan(&metered)
				require.NoError(t, err)
				err = db.QueryRow(`SELECT COUNT(*) FROM rate_cards WHERE entitlement_template_entitlement_type = 'static'`).Scan(&static)
				require.NoError(t, err)
				err = db.QueryRow(`SELECT COUNT(*) FROM rate_cards WHERE entitlement_template_entitlement_type = 'boolean'`).Scan(&boolean)
				require.NoError(t, err)

				// Verify both metered, static and boolean entitlement types were properly migrated
				require.True(t, metered > 0, "Should have metered entitlement types")
				require.True(t, static > 0, "Should have static entitlement types")
				require.True(t, boolean > 0, "Should have boolean entitlement types")

				// Verify plan_rate_card migration details
				fetchAndVerifyRateCard(t, db, "plan-ratecard-1", "plan_rate_cards", "sql_migration_original_plan_rate_card_id", map[string]interface{}{
					"Namespace":       "default",
					"Name":            "Plan RateCard 1",
					"Description":     "Usage Based with Metered",
					"Key":             "plan-ratecard-1",
					"EntitlementType": "metered",
					"Type":            "usage_based",
					"FeatureKey":      "api-calls",
					"BillingCadence":  "P1M",
					"IssueAfterReset": float64(1000),
					"UsagePeriod":     "P1M",
					"HasOriginalID":   true,
					"MetadataContains": map[string]string{
						"EntitlementTemplateMetadata": "test",
					},
				})

				// Verify flat fee plan_rate_card migration details
				fetchAndVerifyRateCard(t, db, "plan-ratecard-2", "plan_rate_cards", "sql_migration_original_plan_rate_card_id", map[string]interface{}{
					"Namespace":       "default",
					"Name":            "Plan RateCard 2",
					"Description":     "Flat Fee with Static",
					"Key":             "plan-ratecard-2",
					"EntitlementType": "static",
					"Type":            "flat_fee",
					"FeatureKey":      "seats",
					"BillingCadence":  "P1M",
					"HasOriginalID":   true,
				})

				// Verify plan_rate_card with existing reference
				fetchAndVerifyRateCard(t, db, "existing-rate-card", "plan_rate_cards", "", map[string]interface{}{
					"Namespace":       "default",
					"Name":            "Existing Rate Card",
					"Key":             "existing-rate-card",
					"EntitlementType": "metered",
					"Type":            "usage_based",
					"FeatureKey":      "existing-feature",
					"HasOriginalID":   false,
				})

				// Verify addon_rate_card_1 migration details
				fetchAndVerifyRateCard(t, db, "addon-ratecard-1", "addon_rate_cards", "sql_migration_original_addon_rate_card_id", map[string]interface{}{
					"Namespace":       "default",
					"Name":            "Addon RateCard 1",
					"Description":     "Usage Based with Boolean",
					"Key":             "addon-ratecard-1",
					"EntitlementType": "boolean",
					"Type":            "usage_based",
					"FeatureKey":      "storage",
					"BillingCadence":  "P1M",
					"HasOriginalID":   true,
				})

				// Verify addon_rate_card with metadata
				fetchAndVerifyRateCard(t, db, "addon-ratecard-2", "addon_rate_cards", "sql_migration_original_addon_rate_card_id", map[string]interface{}{
					"Namespace":       "enterprise",
					"Name":            "Addon RateCard 2",
					"Description":     "Flat Fee with Metadata",
					"Key":             "addon-ratecard-2",
					"EntitlementType": "static",
					"Type":            "flat_fee",
					"FeatureKey":      "premium-support",
					"BillingCadence":  "P1Y",
					"HasOriginalID":   true,
					"MetadataContains": map[string]string{
						"EntitlementTemplateMetadata": "priority",
						"Metadata":                    "premium",
					},
				})

				// Verify subscription_item migration details
				fetchAndVerifyRateCard(t, db, "sub-item-1", "subscription_items", "sql_migration_original_subscription_item_id", map[string]interface{}{
					"Namespace":       "default",
					"Name":            "Sub Item 1",
					"Description":     "With all fields",
					"Key":             "sub-item-1",
					"EntitlementType": "static",
					"Type":            "usage_based",
					"FeatureKey":      "seats",
					"BillingCadence":  "P1M",
					"IsSoftLimit":     true,
					"HasOriginalID":   true,
				})

				// Verify second subscription_item migration details
				fetchAndVerifyRateCard(t, db, "sub-item-2", "subscription_items", "sql_migration_original_subscription_item_id", map[string]interface{}{
					"Namespace":       "default",
					"Name":            "Sub Item 2",
					"Key":             "sub-item-2",
					"EntitlementType": "metered",
					"Type":            "usage_based",
					"FeatureKey":      "api-calls",
					"UsagePeriod":     "P1M",
					"HasOriginalID":   true,
				})

				t.Log("RateCard migration verified successfully")
			},
		},
		{
			// Verify the down migration works correctly
			version:   20250506140848, // The migration after the onewe're testing
			direction: directionDown,
			action: func(t *testing.T, db *sql.DB) {
				t.Log("Verifying rate cards down migration results")

				// Count the number of rate_cards after down migration
				// All rate cards should remain
				var rateCardCount int
				err := db.QueryRow(`SELECT COUNT(*) FROM rate_cards`).Scan(&rateCardCount)
				require.NoError(t, err)
				require.Equal(t, 7, rateCardCount, "Should have 7 rate cards after down migration")

				// Verify that the original rate card is still present
				var existingRateCardName string
				err = db.QueryRow(`SELECT name FROM rate_cards WHERE key = 'existing-rate-card'`).Scan(&existingRateCardName)
				require.NoError(t, err)
				require.Equal(t, "Existing Rate Card", existingRateCardName, "The remaining rate card should be the pre-existing one")

				// Verify that all plan_rate_cards still exist and no longer have ratecard_id references
				var planRateCardsCount, planRateCardsWithRefs int
				err = db.QueryRow(`SELECT COUNT(*) FROM plan_rate_cards`).Scan(&planRateCardsCount)
				require.NoError(t, err)
				require.Equal(t, 3, planRateCardsCount, "All plan_rate_cards should still exist")

				err = db.QueryRow(`SELECT COUNT(*) FROM plan_rate_cards WHERE ratecard_id IS NOT NULL`).Scan(&planRateCardsWithRefs)
				require.NoError(t, err)
				require.Equal(t, 0, planRateCardsWithRefs, "Non-pre-existing plan_rate_cards should no longer have ratecard_id references")

				// Verify that all addon_rate_cards still exist and no longer have ratecard_id references
				var addonRateCardsCount, addonRateCardsWithRefs int
				err = db.QueryRow(`SELECT COUNT(*) FROM addon_rate_cards`).Scan(&addonRateCardsCount)
				require.NoError(t, err)
				require.Equal(t, 2, addonRateCardsCount, "All addon_rate_cards should still exist")

				err = db.QueryRow(`SELECT COUNT(*) FROM addon_rate_cards WHERE ratecard_id IS NOT NULL`).Scan(&addonRateCardsWithRefs)
				require.NoError(t, err)
				require.Equal(t, 0, addonRateCardsWithRefs, "Addon_rate_cards should no longer have ratecard_id references")

				// Verify that all subscription_items still exist and no longer have ratecard_id references
				var subscriptionItemsCount, subscriptionItemsWithRefs int
				err = db.QueryRow(`SELECT COUNT(*) FROM subscription_items`).Scan(&subscriptionItemsCount)
				require.NoError(t, err)
				require.Equal(t, 2, subscriptionItemsCount, "All subscription_items should still exist")

				err = db.QueryRow(`SELECT COUNT(*) FROM subscription_items WHERE ratecard_id IS NOT NULL`).Scan(&subscriptionItemsWithRefs)
				require.NoError(t, err)
				require.Equal(t, 0, subscriptionItemsWithRefs, "Subscription_items should no longer have ratecard_id references")

				// Verify that individual records still have their data intact
				var planRateCardName, planRateCardType, planRateCardFeatureKey string
				err = db.QueryRow(`
					SELECT name, type, feature_key
					FROM plan_rate_cards
					WHERE key = 'plan-ratecard-1'
				`).Scan(&planRateCardName, &planRateCardType, &planRateCardFeatureKey)
				require.NoError(t, err)
				require.Equal(t, "Plan RateCard 1", planRateCardName, "Name should be preserved")
				require.Equal(t, "usage_based", planRateCardType, "Type should be preserved")
				require.Equal(t, "api-calls", planRateCardFeatureKey, "Feature key should be preserved")

				var addonRateCardName, addonRateCardType, addonRateCardFeatureKey string
				err = db.QueryRow(`
					SELECT name, type, feature_key
					FROM addon_rate_cards
					WHERE key = 'addon-ratecard-2'
				`).Scan(&addonRateCardName, &addonRateCardType, &addonRateCardFeatureKey)
				require.NoError(t, err)
				require.Equal(t, "Addon RateCard 2", addonRateCardName, "Name should be preserved")
				require.Equal(t, "flat_fee", addonRateCardType, "Type should be preserved")
				require.Equal(t, "premium-support", addonRateCardFeatureKey, "Feature key should be preserved")

				var subscriptionItemName, subscriptionItemFeatureKey string
				err = db.QueryRow(`
					SELECT name, feature_key
					FROM subscription_items
					WHERE key = 'sub-item-1'
				`).Scan(&subscriptionItemName, &subscriptionItemFeatureKey)
				require.NoError(t, err)
				require.Equal(t, "Sub Item 1", subscriptionItemName, "Name should be preserved")
				require.Equal(t, "seats", subscriptionItemFeatureKey, "Feature key should be preserved")

				t.Log("RateCard down migration verified successfully")
			},
		},
	}}.Test(t)
}
