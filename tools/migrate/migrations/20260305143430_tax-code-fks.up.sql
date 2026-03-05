-- modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "addon_rate_cards_tax_codes_addon_rate_cards" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "billing_customer_overrides" table
ALTER TABLE "billing_customer_overrides" ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "billing_customer_overrides_tax_codes_billing_customer_overrides" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "billing_invoice_lines_tax_codes_billing_invoice_lines" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "billing_invoice_split_line_groups_tax_codes_billing_invoice_spl" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "billing_standard_invoice_detailed_lines" table
ALTER TABLE "billing_standard_invoice_detailed_lines" ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "billing_standard_invoice_detailed_lines_tax_codes_billing_stand" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "billing_workflow_configs_tax_codes_billing_workflow_configs" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "plan_rate_cards_tax_codes_plan_rate_cards" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "subscription_items" table
ALTER TABLE "subscription_items" ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "subscription_items_tax_codes_subscription_items" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
