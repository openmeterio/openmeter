package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestProductCatalogCurrencyReferencesMigration(t *testing.T) {
	const (
		namespace              = "default"
		legacyPlanCurrency     = "USD"
		legacyAddonCurrency    = "GBP"
		legacyOverrideCurrency = "EUR"
	)

	planID := ulid.Make()
	planPhaseID := ulid.Make()
	planInheritedRateCardID := ulid.Make()
	planOverrideRateCardID := ulid.Make()
	addonID := ulid.Make()
	addonInheritedRateCardID := ulid.Make()
	addonOverrideRateCardID := ulid.Make()
	customCurrencyResourceID := ulid.Make()

	runner{
		stops: stops{
			{
				// Populate the last schema where all product-catalog currencies were stored as codes.
				// Before: 20260717143312_add_rate_card_currency.up.sql
				// After: 20260717143818_add_product_catalog_currency_references.up.sql
				version:   20260717143312,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					_, err := db.Exec(`
						INSERT INTO plans (
							namespace,
							id,
							created_at,
							updated_at,
							name,
							key,
							version,
							currency,
							billing_cadence,
							pro_rating_config
						)
						VALUES ($1, $2, NOW(), NOW(), 'Legacy plan', 'legacy_plan', 1, $3, 'P1M', '{"enabled": true, "mode": "prorate_prices"}'::jsonb)
					`, namespace, planID.String(), legacyPlanCurrency)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO plan_phases (
							namespace,
							id,
							created_at,
							updated_at,
							name,
							key,
							plan_id,
							index
						)
						VALUES ($1, $2, NOW(), NOW(), 'Legacy phase', 'legacy_phase', $3, 0)
					`, namespace, planPhaseID.String(), planID.String())
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO plan_rate_cards (
							namespace,
							id,
							created_at,
							updated_at,
							name,
							key,
							type,
							price,
							currency,
							phase_id
						)
						VALUES
							($1, $2, NOW(), NOW(), 'Inherited plan rate card', 'plan_inherited', 'FLAT_FEE', '{"type": "flat", "amount": "10", "paymentTerm": "in_advance"}'::jsonb, NULL, $3),
							($1, $4, NOW(), NOW(), 'Overridden plan rate card', 'plan_override', 'FLAT_FEE', '{"type": "flat", "amount": "20", "paymentTerm": "in_advance"}'::jsonb, $5, $3)
					`,
						namespace,
						planInheritedRateCardID.String(),
						planPhaseID.String(),
						planOverrideRateCardID.String(),
						legacyOverrideCurrency,
					)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO addons (
							namespace,
							id,
							created_at,
							updated_at,
							name,
							key,
							version,
							currency
						)
						VALUES ($1, $2, NOW(), NOW(), 'Legacy add-on', 'legacy_addon', 1, $3)
					`, namespace, addonID.String(), legacyAddonCurrency)
					require.NoError(t, err)

					_, err = db.Exec(`
						INSERT INTO addon_rate_cards (
							namespace,
							id,
							created_at,
							updated_at,
							name,
							key,
							type,
							price,
							currency,
							addon_id
						)
						VALUES
							($1, $2, NOW(), NOW(), 'Inherited add-on rate card', 'addon_inherited', 'FLAT_FEE', '{"type": "flat", "amount": "30", "paymentTerm": "in_advance"}'::jsonb, NULL, $3),
							($1, $4, NOW(), NOW(), 'Overridden add-on rate card', 'addon_override', 'FLAT_FEE', '{"type": "flat", "amount": "40", "paymentTerm": "in_advance"}'::jsonb, $5, $3)
					`,
						namespace,
						addonInheritedRateCardID.String(),
						addonID.String(),
						addonOverrideRateCardID.String(),
						legacyOverrideCurrency,
					)
					require.NoError(t, err)
				},
			},
			{
				// Legacy fiat codes remain code-backed, while nil rate-card codes continue
				// to represent inheritance after managed custom-currency references are added.
				version:   20260717143818,
				direction: directionUp,
				action: func(t *testing.T, db *sql.DB) {
					var (
						fiatCode         sql.NullString
						customCurrencyID sql.NullString
					)

					err := db.QueryRow(`SELECT currency, custom_currency_id FROM plans WHERE id = $1`, planID.String()).Scan(&fiatCode, &customCurrencyID)
					require.NoError(t, err)
					require.Equal(t, legacyPlanCurrency, fiatCode.String)
					require.True(t, fiatCode.Valid)
					require.False(t, customCurrencyID.Valid)

					err = db.QueryRow(`SELECT currency, custom_currency_id FROM addons WHERE id = $1`, addonID.String()).Scan(&fiatCode, &customCurrencyID)
					require.NoError(t, err)
					require.Equal(t, legacyAddonCurrency, fiatCode.String)
					require.True(t, fiatCode.Valid)
					require.False(t, customCurrencyID.Valid)

					err = db.QueryRow(`SELECT currency, custom_currency_id FROM plan_rate_cards WHERE id = $1`, planInheritedRateCardID.String()).Scan(&fiatCode, &customCurrencyID)
					require.NoError(t, err)
					require.False(t, fiatCode.Valid, "a nil legacy override must continue to inherit the plan currency")
					require.False(t, customCurrencyID.Valid)

					err = db.QueryRow(`SELECT currency, custom_currency_id FROM plan_rate_cards WHERE id = $1`, planOverrideRateCardID.String()).Scan(&fiatCode, &customCurrencyID)
					require.NoError(t, err)
					require.Equal(t, legacyOverrideCurrency, fiatCode.String)
					require.True(t, fiatCode.Valid)
					require.False(t, customCurrencyID.Valid)

					err = db.QueryRow(`SELECT currency, custom_currency_id FROM addon_rate_cards WHERE id = $1`, addonInheritedRateCardID.String()).Scan(&fiatCode, &customCurrencyID)
					require.NoError(t, err)
					require.False(t, fiatCode.Valid, "a nil legacy override must continue to inherit the add-on currency")
					require.False(t, customCurrencyID.Valid)

					err = db.QueryRow(`SELECT currency, custom_currency_id FROM addon_rate_cards WHERE id = $1`, addonOverrideRateCardID.String()).Scan(&fiatCode, &customCurrencyID)
					require.NoError(t, err)
					require.Equal(t, legacyOverrideCurrency, fiatCode.String)
					require.True(t, fiatCode.Valid)
					require.False(t, customCurrencyID.Valid)

					var constraintCount int
					err = db.QueryRow(`
						SELECT COUNT(*)
						FROM pg_constraint
						WHERE conname IN (
							'plan_currency_reference',
							'addon_currency_reference',
							'plan_rate_card_currency_reference',
							'plan_rate_card_currency_has_price',
							'addon_rate_card_currency_reference',
							'addon_rate_card_currency_has_price'
						)
					`).Scan(&constraintCount)
					require.NoError(t, err)
					require.Equal(t, 6, constraintCount)

					_, err = db.Exec(`
						INSERT INTO custom_currencies (namespace, id, created_at, updated_at, code, name, symbol)
						VALUES ($1, $2, NOW(), NOW(), 'CREDITS', 'Credits', 'CR')
					`, namespace, customCurrencyResourceID.String())
					require.NoError(t, err)

					_, err = db.Exec(`UPDATE plans SET currency = NULL WHERE id = $1`, planID.String())
					require.Error(t, err, "a top-level currency reference must remain required")

					_, err = db.Exec(`UPDATE plans SET custom_currency_id = $1 WHERE id = $2`, customCurrencyResourceID.String(), planID.String())
					require.Error(t, err, "a top-level currency cannot be both fiat and custom")

					_, err = db.Exec(`UPDATE plan_rate_cards SET custom_currency_id = $1 WHERE id = $2`, customCurrencyResourceID.String(), planOverrideRateCardID.String())
					require.Error(t, err, "a rate-card override cannot be both fiat and custom")

					_, err = db.Exec(`UPDATE plan_rate_cards SET currency = $1, price = NULL WHERE id = $2`, legacyPlanCurrency, planInheritedRateCardID.String())
					require.Error(t, err, "an unpriced rate card must not gain a currency override")
				},
			},
			{
				// Rolling back migrated legacy fiat rows restores the old code-only schema
				// without changing their top-level or explicit override values.
				version:   20260717143312,
				direction: directionDown,
				action: func(t *testing.T, db *sql.DB) {
					var currency string

					err := db.QueryRow(`SELECT currency FROM plans WHERE id = $1`, planID.String()).Scan(&currency)
					require.NoError(t, err)
					require.Equal(t, legacyPlanCurrency, currency)

					err = db.QueryRow(`SELECT currency FROM addons WHERE id = $1`, addonID.String()).Scan(&currency)
					require.NoError(t, err)
					require.Equal(t, legacyAddonCurrency, currency)

					err = db.QueryRow(`SELECT currency FROM plan_rate_cards WHERE id = $1`, planOverrideRateCardID.String()).Scan(&currency)
					require.NoError(t, err)
					require.Equal(t, legacyOverrideCurrency, currency)

					err = db.QueryRow(`SELECT currency FROM addon_rate_cards WHERE id = $1`, addonOverrideRateCardID.String()).Scan(&currency)
					require.NoError(t, err)
					require.Equal(t, legacyOverrideCurrency, currency)

					var customCurrencyColumns int
					err = db.QueryRow(`
						SELECT COUNT(*)
						FROM information_schema.columns
						WHERE table_schema = current_schema()
						  AND table_name IN ('plans', 'addons', 'plan_rate_cards', 'addon_rate_cards')
						  AND column_name = 'custom_currency_id'
					`).Scan(&customCurrencyColumns)
					require.NoError(t, err)
					require.Zero(t, customCurrencyColumns)
				},
			},
		},
	}.Test(t)
}
