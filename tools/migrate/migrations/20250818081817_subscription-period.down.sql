-- reverse: modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" DROP COLUMN "subscription_billing_period_to", DROP COLUMN "subscription_billing_period_from";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP COLUMN "subscription_billing_period_to", DROP COLUMN "subscription_billing_period_from";
