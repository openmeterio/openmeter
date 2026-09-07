-- modify "billing_invoice_validation_issues" table
ALTER TABLE "billing_invoice_validation_issues" ADD COLUMN "attributes" jsonb NULL;
-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ADD COLUMN "validation_issues" jsonb NULL;
-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD COLUMN "validation_issues" jsonb NULL;
-- modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" ADD COLUMN "validation_issues" jsonb NULL;
