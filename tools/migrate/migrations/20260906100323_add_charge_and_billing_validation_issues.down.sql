-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP COLUMN "validation_issues";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP COLUMN "validation_issues";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP COLUMN "validation_issues";
-- reverse: modify "billing_invoice_validation_issues" table
ALTER TABLE "billing_invoice_validation_issues" DROP COLUMN "attributes";
