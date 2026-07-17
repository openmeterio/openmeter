package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionItemCurrencyReferencesMigration(t *testing.T) {
	const namespace = "default"

	customerID := ulid.Make()
	subscriptionID := ulid.Make()
	phaseID := ulid.Make()
	legacyItemID := ulid.Make()
	customCurrencyID := ulid.Make()

	runner{
		stops: stops{
			{
				// Populate the last schema where subscription-item currencies were not persisted.
				// Before: 20260717143818_add_product_catalog_currency_references.up.sql
				// After: 20260717150347_add_subscription_item_currency_references.up.sql
				version:   20260717143818,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					_, err := db.Exec(`
						INSERT INTO customers (
							id,
							namespace,
							created_at,
							updated_at,
							key,
							name,
							currency
						)
						VALUES ($1, $2, NOW(), NOW(), 'legacy-customer', 'Legacy Customer', 'USD')
					`, customerID.String(), namespace)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO subscriptions (
							id,
							namespace,
							created_at,
							updated_at,
							active_from,
							customer_id,
							currency,
							billing_anchor,
							billing_cadence,
							pro_rating_config
						)
						VALUES (
							$1,
							$2,
							NOW(),
							NOW(),
							'2024-01-01 00:00:00',
							$3,
							'USD',
							'2024-01-01 00:00:00',
							'P1M',
							'{"enabled":true,"mode":"prorate_prices"}'::jsonb
						)
					`, subscriptionID.String(), namespace, customerID.String())
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO subscription_phases (
							id,
							namespace,
							created_at,
							updated_at,
							key,
							name,
							subscription_id,
							active_from
						)
						VALUES ($1, $2, NOW(), NOW(), 'default', 'Default', $3, '2024-01-01 00:00:00')
					`, phaseID.String(), namespace, subscriptionID.String())
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO subscription_items (
							id,
							namespace,
							created_at,
							updated_at,
							active_from,
							key,
							name,
							phase_id,
							price
						)
						VALUES (
							$1,
							$2,
							NOW(),
							NOW(),
							'2024-01-01 00:00:00',
							'legacy-priced-item',
							'Legacy priced item',
							$3,
							'{"type":"flat","amount":"10","paymentTerm":"in_advance"}'::jsonb
						)
					`, legacyItemID.String(), namespace, phaseID.String())
					require.NoError(t, err)
				},
			},
			{
				// The expand migration must preserve legacy priced rows without backfilling
				// them, while allowing new writers to store one currency reference.
				version:   20260717150347,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					var (
						fiatCode       sql.NullString
						customCurrency sql.NullString
					)

					err := db.QueryRow(`
						SELECT currency, custom_currency_id
						FROM subscription_items
						WHERE id = $1
					`, legacyItemID.String()).Scan(&fiatCode, &customCurrency)
					require.NoError(t, err)
					require.False(t, fiatCode.Valid, "the expand migration must not backfill the legacy item")
					require.False(t, customCurrency.Valid, "the expand migration must not backfill the legacy item")

					_, err = db.Exec(`
						INSERT INTO custom_currencies (namespace, id, created_at, updated_at, code, name, symbol)
						VALUES ($1, $2, NOW(), NOW(), 'CREDITS', 'Credits', 'CR')
					`, namespace, customCurrencyID.String())
					require.NoError(t, err)

					_, err = db.Exec(`UPDATE subscription_items SET currency = 'USD' WHERE id = $1`, legacyItemID.String())
					require.NoError(t, err, "a priced item must allow a fiat currency reference")

					_, err = db.Exec(`UPDATE subscription_items SET currency = NULL, custom_currency_id = $1 WHERE id = $2`, customCurrencyID.String(), legacyItemID.String())
					require.NoError(t, err, "a priced item must allow a custom currency reference")

					_, err = db.Exec(`UPDATE subscription_items SET currency = 'USD' WHERE id = $1`, legacyItemID.String())
					require.Error(t, err, "a priced item cannot carry both fiat and custom currency references")

					_, err = db.Exec(`UPDATE subscription_items SET custom_currency_id = NULL, currency = 'USD', price = NULL WHERE id = $1`, legacyItemID.String())
					require.Error(t, err, "an unpriced item cannot carry a currency reference")

					var itemCount int
					err = db.QueryRow(`SELECT COUNT(*) FROM subscription_items WHERE id = $1`, legacyItemID.String()).Scan(&itemCount)
					require.NoError(t, err)
					require.Equal(t, 1, itemCount)
				},
			},
			{
				// Rollback removes only the expanded representation and leaves the legacy
				// subscription item and its original price intact.
				version:   20260717143818,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					var price string
					err := db.QueryRow(`SELECT price::text FROM subscription_items WHERE id = $1`, legacyItemID.String()).Scan(&price)
					require.NoError(t, err)
					require.Contains(t, price, `"amount": "10"`)

					var currencyColumns int
					err = db.QueryRow(`
						SELECT COUNT(*)
						FROM information_schema.columns
						WHERE table_schema = current_schema()
						  AND table_name = 'subscription_items'
						  AND column_name IN ('currency', 'custom_currency_id')
					`).Scan(&currencyColumns)
					require.NoError(t, err)
					require.Zero(t, currencyColumns)
				},
			},
		},
	}.Test(t)
}
