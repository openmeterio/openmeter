package migrate_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

const subscriptionItemCurrencyBackfillAnnotation = "dbmigration:backfill_subscription_item_currencies"

type subscriptionItemCurrencyMigrationState struct {
	Currency         sql.NullString
	CustomCurrencyID sql.NullString
	Annotations      map[string]any
}

func querySubscriptionItemCurrencyMigrationStates(t testing.TB, db *sql.DB, phaseID string) map[string]subscriptionItemCurrencyMigrationState {
	t.Helper()

	rows, err := db.QueryContext(t.Context(), `
		SELECT id, currency, custom_currency_id, annotations::text
		FROM subscription_items
		WHERE phase_id = $1
	`, phaseID)
	require.NoError(t, err)
	defer rows.Close()

	states := map[string]subscriptionItemCurrencyMigrationState{}
	for rows.Next() {
		var (
			id              string
			state           subscriptionItemCurrencyMigrationState
			annotationsJSON sql.NullString
		)

		require.NoError(t, rows.Scan(&id, &state.Currency, &state.CustomCurrencyID, &annotationsJSON))
		if annotationsJSON.Valid && annotationsJSON.String != "null" {
			require.NoError(t, json.Unmarshal([]byte(annotationsJSON.String), &state.Annotations))
		}

		states[id] = state
	}
	require.NoError(t, rows.Err())

	return states
}

func TestBackfillSubscriptionItemCurrenciesMigration(t *testing.T) {
	t.Parallel()

	const (
		namespace       = "default"
		previousVersion = uint(20260809172658)
		targetVersion   = uint(20260810064730)
	)

	// given:
	// - legacy priced items with and without annotations, including a soft-deleted item
	// - a legacy priced item already carrying the migration annotation
	// - already materialized fiat and custom-currency items, plus an unpriced item
	// when:
	// - the currency backfill is migrated up and then down
	// then:
	// - only legacy priced items are marked and backfilled, and rollback restores only those rows
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	t.Cleanup(func() { testDB.Close(t) })

	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: testDB.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           testutils.NewLogger(t),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		sourceErr, databaseErr := migrator.Close()
		require.NoError(t, errors.Join(sourceErr, databaseErr))
	})
	require.NoError(t, migrator.Migrate(previousVersion))

	db := testDB.PGDriver.DB()
	customerID := ulid.Make().String()
	subscriptionID := ulid.Make().String()
	phaseID := ulid.Make().String()
	customCurrencyID := ulid.Make().String()
	legacyItemID := ulid.Make().String()
	legacyAnnotatedItemID := ulid.Make().String()
	deletedLegacyItemID := ulid.Make().String()
	alreadyAnnotatedItemID := ulid.Make().String()
	explicitFiatItemID := ulid.Make().String()
	explicitCustomItemID := ulid.Make().String()
	unpricedItemID := ulid.Make().String()

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO customers (id, namespace, created_at, updated_at, key, name, currency)
		VALUES ($1, $2, NOW(), NOW(), 'legacy-customer', 'Legacy Customer', 'USD')
	`, customerID, namespace)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO subscriptions (
			id, namespace, created_at, updated_at, active_from, customer_id, currency,
			billing_anchor, billing_cadence, pro_rating_config
		)
		VALUES (
			$1, $2, NOW(), NOW(), '2024-01-01', $3, 'USD', '2024-01-01', 'P1M',
			'{"enabled":true,"mode":"prorate_prices"}'::jsonb
		)
	`, subscriptionID, namespace, customerID)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO subscription_phases (
			id, namespace, created_at, updated_at, key, name, subscription_id, active_from
		)
		VALUES ($1, $2, NOW(), NOW(), 'default', 'Default', $3, '2024-01-01')
	`, phaseID, namespace, subscriptionID)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO custom_currencies (id, namespace, created_at, updated_at, code, name, symbol)
		VALUES ($1, $2, NOW(), NOW(), 'CREDITS', 'Credits', 'CR')
	`, customCurrencyID, namespace)
	require.NoError(t, err)

	_, err = db.ExecContext(t.Context(), `
		INSERT INTO subscription_items (
			id, namespace, created_at, updated_at, deleted_at, active_from, key, name,
			phase_id, price, currency, custom_currency_id, annotations
		)
		VALUES
			($1, $2, NOW(), NOW(), NULL, '2024-01-01', 'legacy', 'Legacy', $3,
				'{"type":"flat","amount":"10","paymentTerm":"in_advance"}'::jsonb, NULL, NULL, NULL),
			($4, $2, NOW(), NOW(), NULL, '2024-01-01', 'legacy-annotated', 'Legacy Annotated', $3,
				'{"type":"flat","amount":"10","paymentTerm":"in_advance"}'::jsonb, NULL, NULL, '{"existing":"preserved"}'::jsonb),
			($5, $2, NOW(), NOW(), NOW(), '2024-01-01', 'legacy-deleted', 'Legacy Deleted', $3,
				'{"type":"flat","amount":"10","paymentTerm":"in_advance"}'::jsonb, NULL, NULL, NULL),
			($6, $2, NOW(), NOW(), NULL, '2024-01-01', 'explicit-fiat', 'Explicit Fiat', $3,
				'{"type":"flat","amount":"10","paymentTerm":"in_advance"}'::jsonb, 'EUR', NULL, '{"explicit":"fiat"}'::jsonb),
			($7, $2, NOW(), NOW(), NULL, '2024-01-01', 'explicit-custom', 'Explicit Custom', $3,
				'{"type":"flat","amount":"10","paymentTerm":"in_advance"}'::jsonb, 'CREDITS', $8, '{"explicit":"custom"}'::jsonb),
			($9, $2, NOW(), NOW(), NULL, '2024-01-01', 'unpriced', 'Unpriced', $3,
				NULL, NULL, NULL, '{"unpriced":true}'::jsonb),
			($10, $2, NOW(), NOW(), NULL, '2024-01-01', 'already-annotated', 'Already Annotated', $3,
				'{"type":"flat","amount":"10","paymentTerm":"in_advance"}'::jsonb, NULL, NULL,
				'{"dbmigration:backfill_subscription_item_currencies":"2026-08-09T12:00:00Z","existing":"preserved"}'::jsonb)
	`,
		legacyItemID,
		namespace,
		phaseID,
		legacyAnnotatedItemID,
		deletedLegacyItemID,
		explicitFiatItemID,
		explicitCustomItemID,
		customCurrencyID,
		unpricedItemID,
		alreadyAnnotatedItemID,
	)
	require.NoError(t, err)

	require.NoError(t, migrator.Migrate(targetVersion))

	states := querySubscriptionItemCurrencyMigrationStates(t, db, phaseID)
	backfilledItemIDs := []string{legacyItemID, legacyAnnotatedItemID, deletedLegacyItemID}
	var migrationTimestamp string
	for _, itemID := range backfilledItemIDs {
		state := states[itemID]
		require.Equal(t, "USD", state.Currency.String)
		require.False(t, state.CustomCurrencyID.Valid)

		annotation, ok := state.Annotations[subscriptionItemCurrencyBackfillAnnotation]
		require.True(t, ok)
		annotationTimestamp, ok := annotation.(string)
		require.True(t, ok)
		_, err := time.Parse(time.RFC3339Nano, annotationTimestamp)
		require.NoError(t, err)

		if migrationTimestamp == "" {
			migrationTimestamp = annotationTimestamp
		} else {
			require.Equal(t, migrationTimestamp, annotationTimestamp)
		}
	}
	require.Equal(t, "preserved", states[legacyAnnotatedItemID].Annotations["existing"])
	require.False(t, states[alreadyAnnotatedItemID].Currency.Valid)
	require.False(t, states[alreadyAnnotatedItemID].CustomCurrencyID.Valid)
	require.Equal(t, "2026-08-09T12:00:00Z", states[alreadyAnnotatedItemID].Annotations[subscriptionItemCurrencyBackfillAnnotation])
	require.Equal(t, "preserved", states[alreadyAnnotatedItemID].Annotations["existing"])

	require.Equal(t, "EUR", states[explicitFiatItemID].Currency.String)
	require.False(t, states[explicitFiatItemID].CustomCurrencyID.Valid)
	require.Equal(t, "fiat", states[explicitFiatItemID].Annotations["explicit"])
	require.NotContains(t, states[explicitFiatItemID].Annotations, subscriptionItemCurrencyBackfillAnnotation)

	require.Equal(t, "CREDITS", states[explicitCustomItemID].Currency.String)
	require.Equal(t, customCurrencyID, states[explicitCustomItemID].CustomCurrencyID.String)
	require.Equal(t, "custom", states[explicitCustomItemID].Annotations["explicit"])
	require.NotContains(t, states[explicitCustomItemID].Annotations, subscriptionItemCurrencyBackfillAnnotation)

	require.False(t, states[unpricedItemID].Currency.Valid)
	require.False(t, states[unpricedItemID].CustomCurrencyID.Valid)
	require.Equal(t, true, states[unpricedItemID].Annotations["unpriced"])
	require.NotContains(t, states[unpricedItemID].Annotations, subscriptionItemCurrencyBackfillAnnotation)

	require.NoError(t, migrator.Migrate(previousVersion))

	states = querySubscriptionItemCurrencyMigrationStates(t, db, phaseID)
	for _, itemID := range backfilledItemIDs {
		require.False(t, states[itemID].Currency.Valid)
		require.False(t, states[itemID].CustomCurrencyID.Valid)
		require.NotContains(t, states[itemID].Annotations, subscriptionItemCurrencyBackfillAnnotation)
	}
	require.Nil(t, states[legacyItemID].Annotations)
	require.Equal(t, "preserved", states[legacyAnnotatedItemID].Annotations["existing"])
	require.Nil(t, states[deletedLegacyItemID].Annotations)
	require.False(t, states[alreadyAnnotatedItemID].Currency.Valid)
	require.NotContains(t, states[alreadyAnnotatedItemID].Annotations, subscriptionItemCurrencyBackfillAnnotation)
	require.Equal(t, "preserved", states[alreadyAnnotatedItemID].Annotations["existing"])

	require.Equal(t, "EUR", states[explicitFiatItemID].Currency.String)
	require.Equal(t, "CREDITS", states[explicitCustomItemID].Currency.String)
	require.Equal(t, customCurrencyID, states[explicitCustomItemID].CustomCurrencyID.String)
	require.False(t, states[unpricedItemID].Currency.Valid)
}
