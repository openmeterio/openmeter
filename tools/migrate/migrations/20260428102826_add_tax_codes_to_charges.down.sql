-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP CONSTRAINT "charge_usage_based_tax_codes_charge_usage_based", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_tax_codes_charge_flat_fees", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP CONSTRAINT "charge_credit_purchases_tax_codes_charge_credit_purchases", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
