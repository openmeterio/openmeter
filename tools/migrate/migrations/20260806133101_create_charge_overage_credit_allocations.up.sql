-- create "charge_flat_fee_run_overage_credit_allocations" table
CREATE TABLE "charge_flat_fee_run_overage_credit_allocations" (
  "id" character(26) NOT NULL,
  "amount" numeric NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "ledger_transaction_group_id" character(26) NOT NULL,
  "sort_hint" bigint NOT NULL,
  "type" character varying NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "line_id" character(26) NULL,
  "run_id" character(26) NOT NULL,
  "corrects_realization_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_line_charge_ff_overage_credit_alloc" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "charge_ff_overage_credit_alloc_correction" FOREIGN KEY ("corrects_realization_id") REFERENCES "charge_flat_fee_run_overage_credit_allocations" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "charge_ff_overage_credit_alloc_run" FOREIGN KEY ("run_id") REFERENCES "charge_flat_fee_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "chargeflatfeerunoveragecreditallocations_annotations" to table: "charge_flat_fee_run_overage_credit_allocations"
CREATE INDEX "chargeflatfeerunoveragecreditallocations_annotations" ON "charge_flat_fee_run_overage_credit_allocations" USING gin ("annotations");
-- create index "chargeflatfeerunoveragecreditallocations_id" to table: "charge_flat_fee_run_overage_credit_allocations"
CREATE UNIQUE INDEX "chargeflatfeerunoveragecreditallocations_id" ON "charge_flat_fee_run_overage_credit_allocations" ("id");
-- create index "chargeflatfeerunoveragecreditallocations_namespace" to table: "charge_flat_fee_run_overage_credit_allocations"
CREATE INDEX "chargeflatfeerunoveragecreditallocations_namespace" ON "charge_flat_fee_run_overage_credit_allocations" ("namespace");
-- create "charge_usage_based_run_overage_credit_allocations" table
CREATE TABLE "charge_usage_based_run_overage_credit_allocations" (
  "id" character(26) NOT NULL,
  "line_id" character(26) NULL,
  "amount" numeric NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "ledger_transaction_group_id" character(26) NOT NULL,
  "sort_hint" bigint NOT NULL,
  "type" character varying NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "corrects_realization_id" character(26) NULL,
  "run_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_ub_overage_credit_alloc_correction" FOREIGN KEY ("corrects_realization_id") REFERENCES "charge_usage_based_run_overage_credit_allocations" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "charge_ub_overage_credit_alloc_run" FOREIGN KEY ("run_id") REFERENCES "charge_usage_based_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "chargeusagebasedrunoveragecreditallocations_annotations" to table: "charge_usage_based_run_overage_credit_allocations"
CREATE INDEX "chargeusagebasedrunoveragecreditallocations_annotations" ON "charge_usage_based_run_overage_credit_allocations" USING gin ("annotations");
-- create index "chargeusagebasedrunoveragecreditallocations_id" to table: "charge_usage_based_run_overage_credit_allocations"
CREATE UNIQUE INDEX "chargeusagebasedrunoveragecreditallocations_id" ON "charge_usage_based_run_overage_credit_allocations" ("id");
-- create index "chargeusagebasedrunoveragecreditallocations_namespace" to table: "charge_usage_based_run_overage_credit_allocations"
CREATE INDEX "chargeusagebasedrunoveragecreditallocations_namespace" ON "charge_usage_based_run_overage_credit_allocations" ("namespace");
