-- modify "billing_standard_invoice_detailed_lines" table
UPDATE "billing_standard_invoice_detailed_lines"
SET "child_unique_reference_id" = "id"
WHERE "child_unique_reference_id" IS NULL OR "child_unique_reference_id" = '';
-- atlas:nolint MF104
ALTER TABLE "billing_standard_invoice_detailed_lines" ADD CONSTRAINT "child_unique_reference_id_not_empty" CHECK ((child_unique_reference_id)::text <> ''::text), ALTER COLUMN "child_unique_reference_id" SET NOT NULL;
-- modify "charge_flat_fee_detailed_line" table
UPDATE "charge_flat_fee_detailed_line"
SET "child_unique_reference_id" = "id"
WHERE "child_unique_reference_id" IS NULL OR "child_unique_reference_id" = '';
-- atlas:nolint MF104
ALTER TABLE "charge_flat_fee_detailed_line" ADD CONSTRAINT "child_unique_reference_id_not_empty" CHECK ((child_unique_reference_id)::text <> ''::text), ALTER COLUMN "child_unique_reference_id" SET NOT NULL;
-- modify "charge_usage_based_run_detailed_line" table
UPDATE "charge_usage_based_run_detailed_line"
SET "child_unique_reference_id" = "id"
WHERE "child_unique_reference_id" IS NULL OR "child_unique_reference_id" = '';
-- atlas:nolint MF104
ALTER TABLE "charge_usage_based_run_detailed_line" ADD CONSTRAINT "child_unique_reference_id_not_empty" CHECK ((child_unique_reference_id)::text <> ''::text), ALTER COLUMN "child_unique_reference_id" SET NOT NULL;
