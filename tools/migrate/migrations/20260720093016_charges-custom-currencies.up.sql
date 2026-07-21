-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ADD CONSTRAINT "currency_reference" CHECK ((currency IS NULL) <> (custom_currency_id IS NULL)), ALTER COLUMN "currency" DROP NOT NULL, ADD COLUMN "custom_currency_id" character(26) NULL, ADD CONSTRAINT "charge_credit_purchases_custom_currencies_charges_credit_purcha" FOREIGN KEY ("custom_currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD CONSTRAINT "currency_reference" CHECK ((currency IS NULL) <> (custom_currency_id IS NULL)), ALTER COLUMN "currency" DROP NOT NULL, ADD COLUMN "custom_currency_id" character(26) NULL, ADD CONSTRAINT "charge_flat_fees_custom_currencies_charges_flat_fee" FOREIGN KEY ("custom_currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" ADD CONSTRAINT "currency_reference" CHECK ((currency IS NULL) <> (custom_currency_id IS NULL)), ALTER COLUMN "currency" DROP NOT NULL, ADD COLUMN "custom_currency_id" character(26) NULL, ADD CONSTRAINT "charge_usage_based_custom_currencies_charges_usage_based" FOREIGN KEY ("custom_currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;

-- TODO: Add empty checks
