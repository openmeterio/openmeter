-- reverse: modify "billing_invoice_line_discounts" table
ALTER TABLE "billing_invoice_line_discounts" DROP COLUMN "source_discount", DROP COLUMN "pre_line_period_quantity", DROP COLUMN "quantity", DROP COLUMN "rounding_amount", DROP COLUMN "type", ALTER COLUMN "amount" SET NOT NULL;
