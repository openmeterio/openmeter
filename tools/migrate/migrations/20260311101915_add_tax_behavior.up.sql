-- modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" ADD COLUMN "tax_behavior" character varying NULL;
-- modify "billing_customer_overrides" table
ALTER TABLE "billing_customer_overrides" ADD COLUMN "tax_behavior" character varying NULL;
-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "tax_behavior" character varying NULL;
-- modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" ADD COLUMN "tax_behavior" character varying NULL;
-- modify "billing_standard_invoice_detailed_lines" table
ALTER TABLE "billing_standard_invoice_detailed_lines" ADD COLUMN "tax_behavior" character varying NULL;
-- modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" ADD COLUMN "tax_behavior" character varying NULL;
-- modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" ADD COLUMN "tax_behavior" character varying NULL;
-- modify "subscription_items" table
ALTER TABLE "subscription_items" ADD COLUMN "tax_behavior" character varying NULL;
