UPDATE "billing_invoice_lines"
SET "child_unique_reference_id" = 'min-spend'
WHERE "child_unique_reference_id" IS NOT NULL AND "child_unique_reference_id" IN ('unit-price-min-spend', 'volume-min-spend', 'graduated-tiered-min-spend');
