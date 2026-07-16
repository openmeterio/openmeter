-- create "billing_gathering_invoice_lines" table
CREATE TABLE "billing_gathering_invoice_lines" (
  "id" character(26) NOT NULL,
  "annotations" jsonb NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "currency" character varying(3) NOT NULL,
  "service_period_start" timestamptz NOT NULL,
  "service_period_end" timestamptz NOT NULL,
  "tax_config" jsonb NULL,
  "price_type" character varying NOT NULL,
  "feature_key" character varying NULL,
  "price" jsonb NOT NULL,
  "unit_config" jsonb NULL,
  "ratecard_discounts" jsonb NULL,
  "child_unique_reference_id" character varying NULL,
  "subscription_billing_period_from" timestamptz NULL,
  "subscription_billing_period_to" timestamptz NULL,
  "tax_behavior" character varying NULL,
  "invoice_at" timestamptz NOT NULL,
  "managed_by" character varying NOT NULL,
  "engine" character varying NOT NULL DEFAULT 'invoicing',
  "invoice_id" character(26) NOT NULL,
  "split_line_group_id" character(26) NULL,
  "charge_id" character(26) NULL,
  "subscription_id" character(26) NULL,
  "subscription_item_id" character(26) NULL,
  "subscription_phase_id" character(26) NULL,
  "tax_code_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_gathering_line_charge_fk" FOREIGN KEY ("charge_id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "billing_gathering_line_invoice_fk" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "billing_gathering_line_split_group_fk" FOREIGN KEY ("split_line_group_id") REFERENCES "billing_invoice_split_line_groups" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "billing_gathering_line_subscription_fk" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "billing_gathering_line_subscription_item_fk" FOREIGN KEY ("subscription_item_id") REFERENCES "subscription_items" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "billing_gathering_line_subscription_phase_fk" FOREIGN KEY ("subscription_phase_id") REFERENCES "subscription_phases" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "billing_gathering_line_tax_code_fk" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "billinggatheringinvoiceline_annotations" to table: "billing_gathering_invoice_lines"
CREATE INDEX "billinggatheringinvoiceline_annotations" ON "billing_gathering_invoice_lines" USING gin ("annotations");
-- create index "billinggatheringinvoiceline_id" to table: "billing_gathering_invoice_lines"
CREATE UNIQUE INDEX "billinggatheringinvoiceline_id" ON "billing_gathering_invoice_lines" ("id");
-- create index "billinggatheringinvoiceline_namespace" to table: "billing_gathering_invoice_lines"
CREATE INDEX "billinggatheringinvoiceline_namespace" ON "billing_gathering_invoice_lines" ("namespace");
-- create index "billinggatheringinvoiceline_namespace_charge_id" to table: "billing_gathering_invoice_lines"
CREATE INDEX "billinggatheringinvoiceline_namespace_charge_id" ON "billing_gathering_invoice_lines" ("namespace", "charge_id");
-- create index "billinggatheringinvoiceline_namespace_id" to table: "billing_gathering_invoice_lines"
CREATE UNIQUE INDEX "billinggatheringinvoiceline_namespace_id" ON "billing_gathering_invoice_lines" ("namespace", "id");
-- create index "billinggatheringinvoiceline_namespace_invoice_id" to table: "billing_gathering_invoice_lines"
CREATE INDEX "billinggatheringinvoiceline_namespace_invoice_id" ON "billing_gathering_invoice_lines" ("namespace", "invoice_id");
-- create index "billinggatheringinvoiceline_namespace_split_line_group_id" to table: "billing_gathering_invoice_lines"
CREATE INDEX "billinggatheringinvoiceline_namespace_split_line_group_id" ON "billing_gathering_invoice_lines" ("namespace", "split_line_group_id");
-- create index "billinggatheringinvoiceline_tax_code_id" to table: "billing_gathering_invoice_lines"
CREATE INDEX "billinggatheringinvoiceline_tax_code_id" ON "billing_gathering_invoice_lines" ("tax_code_id");
-- create index "billinggatheringline_ns_invoice_child_id" to table: "billing_gathering_invoice_lines"
CREATE UNIQUE INDEX "billinggatheringline_ns_invoice_child_id" ON "billing_gathering_invoice_lines" ("namespace", "invoice_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- create index "billinggatheringline_ns_subscription_ref" to table: "billing_gathering_invoice_lines"
CREATE INDEX "billinggatheringline_ns_subscription_ref" ON "billing_gathering_invoice_lines" ("namespace", "subscription_id", "subscription_phase_id", "subscription_item_id");
