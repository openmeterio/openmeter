-- create "charge_flat_fee_detailed_line" table
CREATE TABLE "charge_flat_fee_detailed_line" (
  "id" character(26) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "tax_config" jsonb NULL,
  "tax_behavior" character varying NULL,
  "service_period_start" timestamptz NOT NULL,
  "service_period_end" timestamptz NOT NULL,
  "quantity" numeric NOT NULL,
  "invoicing_app_external_id" character varying NULL,
  "child_unique_reference_id" character varying NULL,
  "per_unit_amount" numeric NOT NULL,
  "category" character varying NOT NULL DEFAULT 'regular',
  "payment_term" character varying NOT NULL DEFAULT 'in_advance',
  "index" bigint NULL,
  "credits_applied" jsonb NULL,
  "annotations" jsonb NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "amount" numeric NOT NULL,
  "taxes_total" numeric NOT NULL,
  "taxes_inclusive_total" numeric NOT NULL,
  "taxes_exclusive_total" numeric NOT NULL,
  "charges_total" numeric NOT NULL,
  "discounts_total" numeric NOT NULL,
  "credits_total" numeric NOT NULL,
  "total" numeric NOT NULL,
  "charge_id" character(26) NOT NULL,
  "tax_code_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_flat_fee_detailed_line_charge_flat_fees_detailed_lines" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "charge_flat_fee_detailed_line_tax_codes_charge_flat_fee_detaile" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "chargeffdetailedline_ns_charge_child_id" to table: "charge_flat_fee_detailed_line"
CREATE UNIQUE INDEX "chargeffdetailedline_ns_charge_child_id" ON "charge_flat_fee_detailed_line" ("namespace", "charge_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- create index "chargeflatfeedetailedline_annotations" to table: "charge_flat_fee_detailed_line"
CREATE INDEX "chargeflatfeedetailedline_annotations" ON "charge_flat_fee_detailed_line" USING gin ("annotations");
-- create index "chargeflatfeedetailedline_id" to table: "charge_flat_fee_detailed_line"
CREATE UNIQUE INDEX "chargeflatfeedetailedline_id" ON "charge_flat_fee_detailed_line" ("id");
-- create index "chargeflatfeedetailedline_namespace" to table: "charge_flat_fee_detailed_line"
CREATE INDEX "chargeflatfeedetailedline_namespace" ON "charge_flat_fee_detailed_line" ("namespace");
-- create index "chargeflatfeedetailedline_namespace_charge_id" to table: "charge_flat_fee_detailed_line"
CREATE INDEX "chargeflatfeedetailedline_namespace_charge_id" ON "charge_flat_fee_detailed_line" ("namespace", "charge_id");
-- create index "chargeflatfeedetailedline_namespace_id" to table: "charge_flat_fee_detailed_line"
CREATE UNIQUE INDEX "chargeflatfeedetailedline_namespace_id" ON "charge_flat_fee_detailed_line" ("namespace", "id");
-- create index "chargeflatfeedetailedline_tax_code_id" to table: "charge_flat_fee_detailed_line"
CREATE INDEX "chargeflatfeedetailedline_tax_code_id" ON "charge_flat_fee_detailed_line" ("tax_code_id");
-- create "charge_usage_based_detailed_line" table
CREATE TABLE "charge_usage_based_detailed_line" (
  "id" character(26) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "tax_config" jsonb NULL,
  "tax_behavior" character varying NULL,
  "service_period_start" timestamptz NOT NULL,
  "service_period_end" timestamptz NOT NULL,
  "quantity" numeric NOT NULL,
  "invoicing_app_external_id" character varying NULL,
  "child_unique_reference_id" character varying NULL,
  "per_unit_amount" numeric NOT NULL,
  "category" character varying NOT NULL DEFAULT 'regular',
  "payment_term" character varying NOT NULL DEFAULT 'in_advance',
  "index" bigint NULL,
  "credits_applied" jsonb NULL,
  "annotations" jsonb NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "amount" numeric NOT NULL,
  "taxes_total" numeric NOT NULL,
  "taxes_inclusive_total" numeric NOT NULL,
  "taxes_exclusive_total" numeric NOT NULL,
  "charges_total" numeric NOT NULL,
  "discounts_total" numeric NOT NULL,
  "credits_total" numeric NOT NULL,
  "total" numeric NOT NULL,
  "charge_id" character(26) NOT NULL,
  "run_id" character(26) NOT NULL,
  "tax_code_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_usage_based_detailed_line_charge_usage_based_detailed_li" FOREIGN KEY ("charge_id") REFERENCES "charge_usage_based" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "charge_usage_based_detailed_line_charge_usage_based_runs_detail" FOREIGN KEY ("run_id") REFERENCES "charge_usage_based_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "charge_usage_based_detailed_line_tax_codes_charge_usage_based_d" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "chargeubdetailedline_ns_charge_run_child_id" to table: "charge_usage_based_detailed_line"
CREATE UNIQUE INDEX "chargeubdetailedline_ns_charge_run_child_id" ON "charge_usage_based_detailed_line" ("namespace", "charge_id", "run_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- create index "chargeusagebaseddetailedline_annotations" to table: "charge_usage_based_detailed_line"
CREATE INDEX "chargeusagebaseddetailedline_annotations" ON "charge_usage_based_detailed_line" USING gin ("annotations");
-- create index "chargeusagebaseddetailedline_id" to table: "charge_usage_based_detailed_line"
CREATE UNIQUE INDEX "chargeusagebaseddetailedline_id" ON "charge_usage_based_detailed_line" ("id");
-- create index "chargeusagebaseddetailedline_namespace" to table: "charge_usage_based_detailed_line"
CREATE INDEX "chargeusagebaseddetailedline_namespace" ON "charge_usage_based_detailed_line" ("namespace");
-- create index "chargeusagebaseddetailedline_namespace_charge_id" to table: "charge_usage_based_detailed_line"
CREATE INDEX "chargeusagebaseddetailedline_namespace_charge_id" ON "charge_usage_based_detailed_line" ("namespace", "charge_id");
-- create index "chargeusagebaseddetailedline_namespace_id" to table: "charge_usage_based_detailed_line"
CREATE UNIQUE INDEX "chargeusagebaseddetailedline_namespace_id" ON "charge_usage_based_detailed_line" ("namespace", "id");
-- create index "chargeusagebaseddetailedline_namespace_run_id" to table: "charge_usage_based_detailed_line"
CREATE INDEX "chargeusagebaseddetailedline_namespace_run_id" ON "charge_usage_based_detailed_line" ("namespace", "run_id");
-- create index "chargeusagebaseddetailedline_tax_code_id" to table: "charge_usage_based_detailed_line"
CREATE INDEX "chargeusagebaseddetailedline_tax_code_id" ON "charge_usage_based_detailed_line" ("tax_code_id");
