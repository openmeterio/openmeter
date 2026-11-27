-- create "billing_periodic_realization_runs" table
CREATE TABLE "billing_periodic_realization_runs" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "period_to" timestamptz NOT NULL,
  "realize_after" timestamptz NOT NULL,
  "status" character varying NOT NULL,
  "started_at" timestamptz NULL,
  "finished_at" timestamptz NULL,
  "charges_total" bigint NULL,
  "charges_done" bigint NULL,
  "charges_failed" bigint NULL,
  "charges_skipped" bigint NULL,
  "previous_run_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_periodic_realization_run_previous_fk" FOREIGN KEY ("previous_run_id") REFERENCES "billing_periodic_realization_runs" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "billing_periodic_realization_run_realize_after" CHECK (realize_after >= period_to)
);
-- create index "billing_periodic_realization_runs_due" to table: "billing_periodic_realization_runs"
CREATE INDEX "billing_periodic_realization_runs_due" ON "billing_periodic_realization_runs" ("realize_after") WHERE (((status)::text = 'scheduled'::text) AND (deleted_at IS NULL));
-- create index "billing_periodic_realization_runs_one_first" to table: "billing_periodic_realization_runs"
CREATE UNIQUE INDEX "billing_periodic_realization_runs_one_first" ON "billing_periodic_realization_runs" ("namespace") WHERE (previous_run_id IS NULL);
-- create index "billing_periodic_realization_runs_one_running" to table: "billing_periodic_realization_runs"
CREATE UNIQUE INDEX "billing_periodic_realization_runs_one_running" ON "billing_periodic_realization_runs" ("namespace") WHERE ((status)::text = 'running'::text);
-- create index "billing_periodic_realization_runs_previous_run_id_key" to table: "billing_periodic_realization_runs"
CREATE UNIQUE INDEX "billing_periodic_realization_runs_previous_run_id_key" ON "billing_periodic_realization_runs" ("previous_run_id");
-- create index "billingperiodicrealizationrun_id" to table: "billing_periodic_realization_runs"
CREATE UNIQUE INDEX "billingperiodicrealizationrun_id" ON "billing_periodic_realization_runs" ("id");
-- create index "billingperiodicrealizationrun_namespace" to table: "billing_periodic_realization_runs"
CREATE INDEX "billingperiodicrealizationrun_namespace" ON "billing_periodic_realization_runs" ("namespace");
-- create index "billingperiodicrealizationrun_namespace_period_to" to table: "billing_periodic_realization_runs"
CREATE UNIQUE INDEX "billingperiodicrealizationrun_namespace_period_to" ON "billing_periodic_realization_runs" ("namespace", "period_to");
-- create "billing_periodic_realization_charge_states" table
CREATE TABLE "billing_periodic_realization_charge_states" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "customer_id" character(26) NOT NULL,
  "status" character varying NOT NULL,
  "realized_through" timestamptz NULL,
  "target_run_id" character(26) NOT NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_periodic_charge_state_charge_fk" FOREIGN KEY ("charge_id") REFERENCES "charge_usage_based" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "billing_periodic_charge_state_run_fk" FOREIGN KEY ("target_run_id") REFERENCES "billing_periodic_realization_runs" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);
-- create index "billing_periodic_charge_state_run_work" to table: "billing_periodic_realization_charge_states"
CREATE INDEX "billing_periodic_charge_state_run_work" ON "billing_periodic_realization_charge_states" ("namespace", "target_run_id", "status", "customer_id");
-- create index "billing_periodic_realization_charge_states_charge_id_key" to table: "billing_periodic_realization_charge_states"
CREATE UNIQUE INDEX "billing_periodic_realization_charge_states_charge_id_key" ON "billing_periodic_realization_charge_states" ("charge_id");
-- create index "billingperiodicrealizationchargestate_id" to table: "billing_periodic_realization_charge_states"
CREATE UNIQUE INDEX "billingperiodicrealizationchargestate_id" ON "billing_periodic_realization_charge_states" ("id");
-- create index "billingperiodicrealizationchargestate_namespace" to table: "billing_periodic_realization_charge_states"
CREATE INDEX "billingperiodicrealizationchargestate_namespace" ON "billing_periodic_realization_charge_states" ("namespace");
-- create "billing_periodic_realization_run_errors" table
CREATE TABLE "billing_periodic_realization_run_errors" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "code" character varying NOT NULL,
  "reason" text NOT NULL,
  "run_id" character(26) NOT NULL,
  "charge_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_periodic_run_error_charge_fk" FOREIGN KEY ("charge_id") REFERENCES "charge_usage_based" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "billing_periodic_run_error_run_fk" FOREIGN KEY ("run_id") REFERENCES "billing_periodic_realization_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "billingperiodicrealizationrunerror_charge_id" to table: "billing_periodic_realization_run_errors"
CREATE INDEX "billingperiodicrealizationrunerror_charge_id" ON "billing_periodic_realization_run_errors" ("charge_id");
-- create index "billingperiodicrealizationrunerror_id" to table: "billing_periodic_realization_run_errors"
CREATE UNIQUE INDEX "billingperiodicrealizationrunerror_id" ON "billing_periodic_realization_run_errors" ("id");
-- create index "billingperiodicrealizationrunerror_namespace" to table: "billing_periodic_realization_run_errors"
CREATE INDEX "billingperiodicrealizationrunerror_namespace" ON "billing_periodic_realization_run_errors" ("namespace");
-- create index "billingperiodicrealizationrunerror_namespace_run_id" to table: "billing_periodic_realization_run_errors"
CREATE INDEX "billingperiodicrealizationrunerror_namespace_run_id" ON "billing_periodic_realization_run_errors" ("namespace", "run_id");
-- modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "periodic_realization_run_id" character(26) NULL, ADD CONSTRAINT "charge_ub_run_periodic_realization_run_fk" FOREIGN KEY ("periodic_realization_run_id") REFERENCES "billing_periodic_realization_runs" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- create index "charge_ub_run_periodic_realization_run_id" to table: "charge_usage_based_runs"
CREATE INDEX "charge_ub_run_periodic_realization_run_id" ON "charge_usage_based_runs" ("periodic_realization_run_id");
