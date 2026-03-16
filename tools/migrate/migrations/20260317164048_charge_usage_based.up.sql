-- create "charge_usage_based" table
CREATE TABLE "charge_usage_based" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "invoice_at" timestamptz NOT NULL,
  "settlement_mode" character varying NOT NULL,
  "discounts" jsonb NULL,
  "feature_key" character varying NOT NULL,
  "price" jsonb NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_usage_based_charges_usage_based" FOREIGN KEY ("id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "chargeusagebased_id" to table: "charge_usage_based"
CREATE UNIQUE INDEX "chargeusagebased_id" ON "charge_usage_based" ("id");
-- create index "chargeusagebased_namespace" to table: "charge_usage_based"
CREATE INDEX "chargeusagebased_namespace" ON "charge_usage_based" ("namespace");
-- create index "chargeusagebased_namespace_id" to table: "charge_usage_based"
CREATE UNIQUE INDEX "chargeusagebased_namespace_id" ON "charge_usage_based" ("namespace", "id");
-- create "charge_usage_based_runs" table
CREATE TABLE "charge_usage_based_runs" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "type" character varying NOT NULL,
  "asof" timestamptz NOT NULL,
  "meter_value" numeric NOT NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_usage_based_runs_charge_usage_based_runs" FOREIGN KEY ("charge_id") REFERENCES "charge_usage_based" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "chargeusagebasedruns_id" to table: "charge_usage_based_runs"
CREATE UNIQUE INDEX "chargeusagebasedruns_id" ON "charge_usage_based_runs" ("id");
-- create index "chargeusagebasedruns_namespace" to table: "charge_usage_based_runs"
CREATE INDEX "chargeusagebasedruns_namespace" ON "charge_usage_based_runs" ("namespace");
-- create index "chargeusagebasedruns_namespace_charge_id" to table: "charge_usage_based_runs"
CREATE INDEX "chargeusagebasedruns_namespace_charge_id" ON "charge_usage_based_runs" ("namespace", "charge_id");
-- create "charge_usage_based_run_credit_allocations" table
CREATE TABLE "charge_usage_based_run_credit_allocations" (
  "id" character(26) NOT NULL,
  "line_id" character(26) NULL,
  "amount" numeric NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "ledger_transaction_group_id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "run_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_usage_based_run_credit_allocations_charge_usage_based_ru" FOREIGN KEY ("run_id") REFERENCES "charge_usage_based_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "chargeusagebasedruncreditallocations_annotations" to table: "charge_usage_based_run_credit_allocations"
CREATE INDEX "chargeusagebasedruncreditallocations_annotations" ON "charge_usage_based_run_credit_allocations" USING gin ("annotations");
-- create index "chargeusagebasedruncreditallocations_id" to table: "charge_usage_based_run_credit_allocations"
CREATE UNIQUE INDEX "chargeusagebasedruncreditallocations_id" ON "charge_usage_based_run_credit_allocations" ("id");
-- create index "chargeusagebasedruncreditallocations_namespace" to table: "charge_usage_based_run_credit_allocations"
CREATE INDEX "chargeusagebasedruncreditallocations_namespace" ON "charge_usage_based_run_credit_allocations" ("namespace");
-- create "charge_usage_based_run_invoiced_usages" table
CREATE TABLE "charge_usage_based_run_invoiced_usages" (
  "id" character(26) NOT NULL,
  "line_id" character(26) NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "mutable" boolean NOT NULL,
  "ledger_transaction_group_id" character(26) NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "amount" numeric NOT NULL,
  "taxes_total" numeric NOT NULL,
  "taxes_inclusive_total" numeric NOT NULL,
  "taxes_exclusive_total" numeric NOT NULL,
  "charges_total" numeric NOT NULL,
  "discounts_total" numeric NOT NULL,
  "credits_total" numeric NOT NULL,
  "total" numeric NOT NULL,
  "run_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_usage_based_run_invoiced_usages_charge_usage_based_runs_" FOREIGN KEY ("run_id") REFERENCES "charge_usage_based_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "charge_usage_based_run_invoiced_usages_run_id_key" to table: "charge_usage_based_run_invoiced_usages"
CREATE UNIQUE INDEX "charge_usage_based_run_invoiced_usages_run_id_key" ON "charge_usage_based_run_invoiced_usages" ("run_id");
-- create index "chargeusagebasedruninvoicedusage_annotations" to table: "charge_usage_based_run_invoiced_usages"
CREATE INDEX "chargeusagebasedruninvoicedusage_annotations" ON "charge_usage_based_run_invoiced_usages" USING gin ("annotations");
-- create index "chargeusagebasedruninvoicedusage_id" to table: "charge_usage_based_run_invoiced_usages"
CREATE UNIQUE INDEX "chargeusagebasedruninvoicedusage_id" ON "charge_usage_based_run_invoiced_usages" ("id");
-- create index "chargeusagebasedruninvoicedusage_namespace" to table: "charge_usage_based_run_invoiced_usages"
CREATE INDEX "chargeusagebasedruninvoicedusage_namespace" ON "charge_usage_based_run_invoiced_usages" ("namespace");
-- create "charge_usage_based_run_payments" table
CREATE TABLE "charge_usage_based_run_payments" (
  "id" character(26) NOT NULL,
  "line_id" character(26) NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "status" character varying NOT NULL,
  "amount" numeric NOT NULL,
  "authorized_transaction_group_id" character(26) NULL,
  "authorized_at" timestamptz NULL,
  "settled_transaction_group_id" character(26) NULL,
  "settled_at" timestamptz NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "run_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_usage_based_run_payments_charge_usage_based_runs_payment" FOREIGN KEY ("run_id") REFERENCES "charge_usage_based_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "charge_usage_based_run_payments_run_id_key" to table: "charge_usage_based_run_payments"
CREATE UNIQUE INDEX "charge_usage_based_run_payments_run_id_key" ON "charge_usage_based_run_payments" ("run_id");
-- create index "chargeusagebasedrunpayment_annotations" to table: "charge_usage_based_run_payments"
CREATE INDEX "chargeusagebasedrunpayment_annotations" ON "charge_usage_based_run_payments" USING gin ("annotations");
-- create index "chargeusagebasedrunpayment_id" to table: "charge_usage_based_run_payments"
CREATE UNIQUE INDEX "chargeusagebasedrunpayment_id" ON "charge_usage_based_run_payments" ("id");
-- create index "chargeusagebasedrunpayment_namespace" to table: "charge_usage_based_run_payments"
CREATE INDEX "chargeusagebasedrunpayment_namespace" ON "charge_usage_based_run_payments" ("namespace");
