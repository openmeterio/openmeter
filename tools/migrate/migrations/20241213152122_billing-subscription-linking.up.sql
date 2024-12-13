-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "subscription_id" character(26) NULL, ADD COLUMN "subscription_item_id" character(26) NULL, ADD COLUMN "subscription_phase_id" character(26) NULL, ADD
 CONSTRAINT "billing_invoice_lines_subscription_items_billing_lines" FOREIGN KEY ("subscription_item_id") REFERENCES "subscription_items" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD
 CONSTRAINT "billing_invoice_lines_subscription_phases_billing_lines" FOREIGN KEY ("subscription_phase_id") REFERENCES "subscription_phases" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD
 CONSTRAINT "billing_invoice_lines_subscriptions_billing_lines" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "billinginvoiceline_namespace_subscription_id_subscription_phase" to table: "billing_invoice_lines"
CREATE INDEX "billinginvoiceline_namespace_subscription_id_subscription_phase" ON "billing_invoice_lines" ("namespace", "subscription_id", "subscription_phase_id", "subscription_item_id");
