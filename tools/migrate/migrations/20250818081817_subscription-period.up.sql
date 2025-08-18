-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "subscription_billing_period_from" timestamptz NULL, ADD COLUMN "subscription_billing_period_to" timestamptz NULL;
-- modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" ADD COLUMN "subscription_billing_period_from" timestamptz NULL, ADD COLUMN "subscription_billing_period_to" timestamptz NULL;

-- backfill data (so that we have some data on the old invoices), gathering invoices will be updated by the subscription sync
UPDATE billing_invoice_lines
    SET subscription_billing_period_from = period_start,
        subscription_billing_period_to = period_end
    WHERE subscription_id IS NOT NULL;

UPDATE billing_invoice_split_line_groups
    SET subscription_billing_period_from = period_start,
        subscription_billing_period_to = period_end
    WHERE subscription_id IS NOT NULL;
