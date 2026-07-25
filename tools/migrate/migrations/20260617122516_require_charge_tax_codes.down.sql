-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP CONSTRAINT "charge_usage_based_tax_codes_charge_usage_based", ALTER COLUMN "tax_code_id" DROP NOT NULL, ADD CONSTRAINT "charge_usage_based_tax_codes_charge_usage_based" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_tax_codes_charge_flat_fees", ALTER COLUMN "tax_code_id" DROP NOT NULL, ADD CONSTRAINT "charge_flat_fees_tax_codes_charge_flat_fees" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP CONSTRAINT "charge_credit_purchases_tax_codes_charge_credit_purchases", ALTER COLUMN "tax_code_id" DROP NOT NULL, ADD CONSTRAINT "charge_credit_purchases_tax_codes_charge_credit_purchases" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
