-- modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "addon_rate_cards_tax_codes_addon_rate_cards" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL NOT VALID;
-- create index "addonratecard_tax_code_id" to table: "addon_rate_cards"
CREATE INDEX "addonratecard_tax_code_id" ON "addon_rate_cards" ("tax_code_id");
ALTER TABLE "addon_rate_cards" VALIDATE CONSTRAINT "addon_rate_cards_tax_codes_addon_rate_cards";
-- modify "billing_customer_overrides" table
ALTER TABLE "billing_customer_overrides" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "billing_customer_overrides_tax_codes_billing_customer_overrides" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL NOT VALID;
-- create index "billingcustomeroverride_tax_code_id" to table: "billing_customer_overrides"
CREATE INDEX "billingcustomeroverride_tax_code_id" ON "billing_customer_overrides" ("tax_code_id");
ALTER TABLE "billing_customer_overrides" VALIDATE CONSTRAINT "billing_customer_overrides_tax_codes_billing_customer_overrides";
-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "billing_invoice_lines_tax_codes_billing_invoice_lines" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL NOT VALID;
-- create index "billinginvoiceline_tax_code_id" to table: "billing_invoice_lines"
CREATE INDEX "billinginvoiceline_tax_code_id" ON "billing_invoice_lines" ("tax_code_id");
ALTER TABLE "billing_invoice_lines" VALIDATE CONSTRAINT "billing_invoice_lines_tax_codes_billing_invoice_lines";
-- modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "billing_invoice_split_line_groups_tax_codes_billing_invoice_spl" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL NOT VALID;
-- create index "billinginvoicesplitlinegroup_tax_code_id" to table: "billing_invoice_split_line_groups"
CREATE INDEX "billinginvoicesplitlinegroup_tax_code_id" ON "billing_invoice_split_line_groups" ("tax_code_id");
ALTER TABLE "billing_invoice_split_line_groups" VALIDATE CONSTRAINT "billing_invoice_split_line_groups_tax_codes_billing_invoice_spl";
-- modify "billing_standard_invoice_detailed_lines" table
ALTER TABLE "billing_standard_invoice_detailed_lines" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "billing_standard_invoice_detailed_lines_tax_codes_billing_stand" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL NOT VALID;
-- create index "billingstandardinvoicedetailedline_tax_code_id" to table: "billing_standard_invoice_detailed_lines"
CREATE INDEX "billingstandardinvoicedetailedline_tax_code_id" ON "billing_standard_invoice_detailed_lines" ("tax_code_id");
ALTER TABLE "billing_standard_invoice_detailed_lines" VALIDATE CONSTRAINT "billing_standard_invoice_detailed_lines_tax_codes_billing_stand";
-- modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "billing_workflow_configs_tax_codes_billing_workflow_configs" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL NOT VALID;
-- create index "billingworkflowconfig_tax_code_id" to table: "billing_workflow_configs"
CREATE INDEX "billingworkflowconfig_tax_code_id" ON "billing_workflow_configs" ("tax_code_id");
ALTER TABLE "billing_workflow_configs" VALIDATE CONSTRAINT "billing_workflow_configs_tax_codes_billing_workflow_configs";
-- modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "plan_rate_cards_tax_codes_plan_rate_cards" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL NOT VALID;
-- create index "planratecard_tax_code_id" to table: "plan_rate_cards"
CREATE INDEX "planratecard_tax_code_id" ON "plan_rate_cards" ("tax_code_id");
ALTER TABLE "plan_rate_cards" VALIDATE CONSTRAINT "plan_rate_cards_tax_codes_plan_rate_cards";
-- modify "subscription_items" table
ALTER TABLE "subscription_items" ADD COLUMN "tax_behavior" character varying NULL, ADD COLUMN "tax_code_id" character(26) NULL, ADD CONSTRAINT "subscription_items_tax_codes_subscription_items" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL NOT VALID;
-- create index "subscriptionitem_tax_code_id" to table: "subscription_items"
CREATE INDEX "subscriptionitem_tax_code_id" ON "subscription_items" ("tax_code_id");
ALTER TABLE "subscription_items" VALIDATE CONSTRAINT "subscription_items_tax_codes_subscription_items";
