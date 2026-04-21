-- reverse: modify "charge_usage_based_run_detailed_line" table
ALTER TABLE "charge_usage_based_run_detailed_line" ALTER COLUMN "child_unique_reference_id" DROP NOT NULL, DROP CONSTRAINT IF EXISTS "child_unique_reference_id_not_empty";
-- reverse: modify "charge_flat_fee_detailed_line" table
ALTER TABLE "charge_flat_fee_detailed_line" ALTER COLUMN "child_unique_reference_id" DROP NOT NULL, DROP CONSTRAINT IF EXISTS "child_unique_reference_id_not_empty";
-- reverse: modify "billing_standard_invoice_detailed_lines" table
ALTER TABLE "billing_standard_invoice_detailed_lines" ALTER COLUMN "child_unique_reference_id" DROP NOT NULL, DROP CONSTRAINT IF EXISTS "child_unique_reference_id_not_empty";
