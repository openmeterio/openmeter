-- reverse: modify "charge_usage_based_run_detailed_line" table
ALTER TABLE "charge_usage_based_run_detailed_line" ALTER COLUMN "currency" SET NOT NULL;
-- reverse: modify "charge_flat_fee_run_detailed_lines" table
ALTER TABLE "charge_flat_fee_run_detailed_lines" ALTER COLUMN "currency" SET NOT NULL;
-- reverse: modify "billing_standard_invoice_detailed_lines" table
ALTER TABLE "billing_standard_invoice_detailed_lines" ALTER COLUMN "currency" SET NOT NULL;
