-- reverse: create index "billinginvoiceline_namespace_subscription_id_subscription_phase" to table: "billing_invoice_lines"
DROP INDEX "billinginvoiceline_namespace_subscription_id_subscription_phase";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_subscriptions_billing_lines", DROP CONSTRAINT "billing_invoice_lines_subscription_phases_billing_lines", DROP CONSTRAINT "billing_invoice_lines_subscription_items_billing_lines", DROP COLUMN "subscription_phase_id", DROP COLUMN "subscription_item_id", DROP COLUMN "subscription_id";
