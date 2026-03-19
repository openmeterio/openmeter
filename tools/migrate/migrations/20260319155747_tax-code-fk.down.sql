-- reverse: create index "subscriptionitem_tax_code_id" to table: "subscription_items"
DROP INDEX "subscriptionitem_tax_code_id";
-- reverse: modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_items_tax_codes_subscription_items", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
-- reverse: create index "planratecard_tax_code_id" to table: "plan_rate_cards"
DROP INDEX "planratecard_tax_code_id";
-- reverse: modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" DROP CONSTRAINT "plan_rate_cards_tax_codes_plan_rate_cards", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
-- reverse: create index "billingworkflowconfig_tax_code_id" to table: "billing_workflow_configs"
DROP INDEX "billingworkflowconfig_tax_code_id";
-- reverse: modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" DROP CONSTRAINT "billing_workflow_configs_tax_codes_billing_workflow_configs", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
-- reverse: create index "billingstandardinvoicedetailedline_tax_code_id" to table: "billing_standard_invoice_detailed_lines"
DROP INDEX "billingstandardinvoicedetailedline_tax_code_id";
-- reverse: modify "billing_standard_invoice_detailed_lines" table
ALTER TABLE "billing_standard_invoice_detailed_lines" DROP CONSTRAINT "billing_standard_invoice_detailed_lines_tax_codes_billing_stand", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
-- reverse: create index "billinginvoicesplitlinegroup_tax_code_id" to table: "billing_invoice_split_line_groups"
DROP INDEX "billinginvoicesplitlinegroup_tax_code_id";
-- reverse: modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" DROP CONSTRAINT "billing_invoice_split_line_groups_tax_codes_billing_invoice_spl", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
-- reverse: create index "billinginvoiceline_tax_code_id" to table: "billing_invoice_lines"
DROP INDEX "billinginvoiceline_tax_code_id";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_tax_codes_billing_invoice_lines", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
-- reverse: create index "billingcustomeroverride_tax_code_id" to table: "billing_customer_overrides"
DROP INDEX "billingcustomeroverride_tax_code_id";
-- reverse: modify "billing_customer_overrides" table
ALTER TABLE "billing_customer_overrides" DROP CONSTRAINT "billing_customer_overrides_tax_codes_billing_customer_overrides", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
-- reverse: create index "addonratecard_tax_code_id" to table: "addon_rate_cards"
DROP INDEX "addonratecard_tax_code_id";
-- reverse: modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" DROP CONSTRAINT "addon_rate_cards_tax_codes_addon_rate_cards", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
