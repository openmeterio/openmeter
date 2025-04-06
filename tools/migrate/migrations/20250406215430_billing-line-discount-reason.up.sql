-- modify "billing_invoice_line_discounts" table
ALTER TABLE "billing_invoice_line_discounts" ADD COLUMN "reason" character varying NOT NULL DEFAULT 'maximum_spend';
ALTER TABLE "billing_invoice_line_discounts" ALTER COLUMN "reason" DROP DEFAULT;
