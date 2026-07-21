-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP CONSTRAINT "charge_usage_based_custom_currencies_charges_usage_based", DROP COLUMN "custom_currency_id", ALTER COLUMN "currency" SET NOT NULL, DROP CONSTRAINT "currency_reference";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_custom_currencies_charges_flat_fee", DROP COLUMN "custom_currency_id", ALTER COLUMN "currency" SET NOT NULL, DROP CONSTRAINT "currency_reference";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP CONSTRAINT "charge_credit_purchases_custom_currencies_charges_credit_purcha", DROP COLUMN "custom_currency_id", ALTER COLUMN "currency" SET NOT NULL, DROP CONSTRAINT "currency_reference";
