-- Restore the empty storage schema only; retain the JSON archive for recovery.
-- reverse: drop "billing_invoice_line_discounts" table
CREATE TABLE "billing_invoice_line_discounts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "child_unique_reference_id" character varying NULL,
  "description" character varying NULL,
  "amount" numeric NOT NULL,
  "line_id" character(26) NOT NULL,
  "invoicing_app_external_id" character varying NULL,
  "reason" character varying NOT NULL,
  "rounding_amount" numeric NULL,
  "source_discount" jsonb NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_invoice_line_discounts_billing_invoice_lines_line_amoun" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
CREATE INDEX "billing_invoice_line_discounts_line_id_idx" ON "billing_invoice_line_discounts" ("line_id");
CREATE UNIQUE INDEX "billinginvoicelinediscount_id" ON "billing_invoice_line_discounts" ("id");
CREATE INDEX "billinginvoicelinediscount_namespace" ON "billing_invoice_line_discounts" ("namespace");
CREATE INDEX "billinginvoicelinediscount_namespace_line_id" ON "billing_invoice_line_discounts" ("namespace", "line_id");
CREATE UNIQUE INDEX "billinginvoicelinediscount_namespace_line_id_child_unique_refer" ON "billing_invoice_line_discounts" ("namespace", "line_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- reverse: drop "billing_invoice_flat_fee_line_configs" table
CREATE TABLE "billing_invoice_flat_fee_line_configs" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "per_unit_amount" numeric NOT NULL,
  "category" character varying NOT NULL DEFAULT 'regular',
  "payment_term" character varying NOT NULL DEFAULT 'in_advance',
  "index" bigint NULL,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "billinginvoiceflatfeelineconfig_id" ON "billing_invoice_flat_fee_line_configs" ("id");
CREATE INDEX "billinginvoiceflatfeelineconfig_namespace" ON "billing_invoice_flat_fee_line_configs" ("namespace");
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "fee_line_config_id" character(26) NULL;
CREATE INDEX "billing_invoice_lines_fee_config_id_idx" ON "billing_invoice_lines" ("fee_line_config_id");
