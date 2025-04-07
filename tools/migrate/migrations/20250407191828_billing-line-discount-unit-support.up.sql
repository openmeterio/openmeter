-- modify "billing_invoice_line_discounts" table
ALTER TABLE "billing_invoice_line_discounts" ALTER COLUMN "amount" DROP NOT NULL, ADD COLUMN "type" character varying NOT NULL DEFAULT 'amount', ADD COLUMN "rounding_amount" numeric NULL DEFAULT 0, ADD COLUMN "quantity" numeric NULL, ADD COLUMN "pre_line_period_quantity" numeric NULL, ADD COLUMN "source_discount" jsonb NULL;

ALTER TABLE "billing_invoice_line_discounts" ALTER COLUMN "type" DROP DEFAULT;
ALTER TABLE "billing_invoice_line_discounts" ALTER COLUMN "rounding_amount" DROP DEFAULT;
