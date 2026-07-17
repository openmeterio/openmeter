package migrate_test

import (
	"errors"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestSubscriptionCustomCurrencySemanticsMigrationRequiresBackfill(t *testing.T) {
	const (
		namespace       = "default"
		previousVersion = uint(20260717150347)
		targetVersion   = uint(20260717195001)
	)

	// given:
	// - an expand-and-write schema containing one legacy priced item without currency
	// when:
	// - the strict semantic migration is attempted before and after the documented backfill
	// then:
	// - it first fails safely, then installs strict item and pinned-cost-basis invariants
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
	itemID := ulid.Make().String()

	_, err = db.Exec(`
		INSERT INTO customers (id, namespace, created_at, updated_at, key, name, currency)
		VALUES ($1, $2, NOW(), NOW(), 'legacy-customer', 'Legacy Customer', 'USD')
	`, customerID, namespace)
	require.NoError(t, err)

	_, err = db.Exec(`
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

	_, err = db.Exec(`
		INSERT INTO subscription_phases (
			id, namespace, created_at, updated_at, key, name, subscription_id, active_from
		)
		VALUES ($1, $2, NOW(), NOW(), 'default', 'Default', $3, '2024-01-01')
	`, phaseID, namespace, subscriptionID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO subscription_items (
			id, namespace, created_at, updated_at, active_from, key, name, phase_id, price
		)
		VALUES (
			$1, $2, NOW(), NOW(), '2024-01-01', 'legacy', 'Legacy', $3,
			'{"type":"flat","amount":"10","paymentTerm":"in_advance"}'::jsonb
		)
	`, itemID, namespace, phaseID)
	require.NoError(t, err)

	err = migrator.Migrate(targetVersion)
	require.Error(t, err, "strict migration must not silently strand a legacy priced item")
	require.NoError(t, migrator.Force(previousVersion), "clear the failed migration marker after the transactional DDL rollback")

	_, err = db.Exec(`
		UPDATE subscription_items AS item
		SET currency = subscription.currency,
		    updated_at = NOW()
		FROM subscription_phases AS phase
		JOIN subscriptions AS subscription
		  ON subscription.id = phase.subscription_id
		 AND subscription.namespace = phase.namespace
		WHERE item.phase_id = phase.id
		  AND item.namespace = phase.namespace
		  AND item.price IS NOT NULL
		  AND item.currency IS NULL
		  AND item.custom_currency_id IS NULL
	`)
	require.NoError(t, err)
	require.NoError(t, migrator.Migrate(targetVersion))

	var costBasisMode string
	err = db.QueryRow(`SELECT cost_basis_mode FROM subscriptions WHERE id = $1`, subscriptionID).Scan(&costBasisMode)
	require.NoError(t, err)
	require.Equal(t, "dynamic", costBasisMode)

	var subscriptionLookupIndex *string
	err = db.QueryRow(`SELECT to_regclass('subscriptioncostbasispin_subscription_id')::text`).Scan(&subscriptionLookupIndex)
	require.NoError(t, err)
	require.NotNil(t, subscriptionLookupIndex, "pin eager-loading must have a subscription ID index")

	_, err = db.Exec(`UPDATE subscription_items SET currency = NULL WHERE id = $1`, itemID)
	require.Error(t, err, "priced items must carry exactly one currency reference")

	_, err = db.Exec(`UPDATE subscription_items SET price = NULL WHERE id = $1`, itemID)
	require.Error(t, err, "unpriced items must not carry a currency reference")

	customCurrencyID := ulid.Make().String()
	costBasisID := ulid.Make().String()
	pinID := ulid.Make().String()
	_, err = db.Exec(`
		INSERT INTO custom_currencies (id, namespace, created_at, updated_at, code, name, symbol)
		VALUES ($1, $2, NOW(), NOW(), 'CREDITS', 'Credits', 'CR')
	`, customCurrencyID, namespace)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO currency_cost_bases (
			id, namespace, created_at, updated_at, currency_id, fiat_code, rate, effective_from
		)
		VALUES ($1, $2, NOW(), NOW(), $3, 'USD', 0.5, '2024-01-01')
	`, costBasisID, namespace, customCurrencyID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO subscription_cost_basis_pins (
			id, namespace, created_at, updated_at, subscription_id, custom_currency_id,
			invoice_currency, cost_basis_id
		)
		VALUES ($1, $2, NOW(), NOW(), $3, $4, 'USD', $5)
	`, pinID, namespace, subscriptionID, customCurrencyID, costBasisID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO subscription_cost_basis_pins (
			id, namespace, created_at, updated_at, subscription_id, custom_currency_id,
			invoice_currency, cost_basis_id
		)
		VALUES ($1, $2, NOW(), NOW(), $3, $4, 'USD', $5)
	`, ulid.Make().String(), namespace, subscriptionID, customCurrencyID, costBasisID)
	require.Error(t, err, "a subscription can pin a custom/fiat pair only once")

	var pinCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM subscription_cost_basis_pins WHERE id = $1`, pinID).Scan(&pinCount)
	require.NoError(t, err)
	require.Equal(t, 1, pinCount)

	require.NoError(t, migrator.Migrate(previousVersion))
	var pinTable *string
	err = db.QueryRow(`SELECT to_regclass('subscription_cost_basis_pins')::text`).Scan(&pinTable)
	require.NoError(t, err)
	require.Nil(t, pinTable)
}
