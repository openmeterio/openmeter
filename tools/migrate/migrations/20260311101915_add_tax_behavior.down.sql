-- reverse: modify "subscription_items" table
ALTER TABLE "subscription_items" DROP COLUMN "tax_behavior";
-- reverse: modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" DROP COLUMN "tax_behavior";
-- reverse: modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" DROP COLUMN "tax_behavior";
-- reverse: modify "billing_standard_invoice_detailed_lines" table
ALTER TABLE "billing_standard_invoice_detailed_lines" DROP COLUMN "tax_behavior";
-- reverse: modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" DROP COLUMN "tax_behavior";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP COLUMN "tax_behavior";
-- reverse: modify "billing_customer_overrides" table
ALTER TABLE "billing_customer_overrides" DROP COLUMN "tax_behavior";
-- reverse: modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" DROP COLUMN "tax_behavior";
