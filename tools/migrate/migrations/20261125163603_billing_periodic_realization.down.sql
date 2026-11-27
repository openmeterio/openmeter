-- reverse: create index "charge_ub_run_periodic_realization_run_id" to table: "charge_usage_based_runs"
DROP INDEX "charge_ub_run_periodic_realization_run_id";
-- reverse: modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" DROP CONSTRAINT "charge_ub_run_periodic_realization_run_fk", DROP COLUMN "periodic_realization_run_id";
-- reverse: create index "billingperiodicrealizationrunerror_namespace_run_id" to table: "billing_periodic_realization_run_errors"
DROP INDEX "billingperiodicrealizationrunerror_namespace_run_id";
-- reverse: create index "billingperiodicrealizationrunerror_namespace" to table: "billing_periodic_realization_run_errors"
DROP INDEX "billingperiodicrealizationrunerror_namespace";
-- reverse: create index "billingperiodicrealizationrunerror_id" to table: "billing_periodic_realization_run_errors"
DROP INDEX "billingperiodicrealizationrunerror_id";
-- reverse: create index "billingperiodicrealizationrunerror_charge_id" to table: "billing_periodic_realization_run_errors"
DROP INDEX "billingperiodicrealizationrunerror_charge_id";
-- reverse: create "billing_periodic_realization_run_errors" table
DROP TABLE "billing_periodic_realization_run_errors";
-- reverse: create index "billingperiodicrealizationchargestate_namespace" to table: "billing_periodic_realization_charge_states"
DROP INDEX "billingperiodicrealizationchargestate_namespace";
-- reverse: create index "billingperiodicrealizationchargestate_id" to table: "billing_periodic_realization_charge_states"
DROP INDEX "billingperiodicrealizationchargestate_id";
-- reverse: create index "billing_periodic_realization_charge_states_charge_id_key" to table: "billing_periodic_realization_charge_states"
DROP INDEX "billing_periodic_realization_charge_states_charge_id_key";
-- reverse: create index "billing_periodic_charge_state_run_work" to table: "billing_periodic_realization_charge_states"
DROP INDEX "billing_periodic_charge_state_run_work";
-- reverse: create "billing_periodic_realization_charge_states" table
DROP TABLE "billing_periodic_realization_charge_states";
-- reverse: create index "billingperiodicrealizationrun_namespace_period_to" to table: "billing_periodic_realization_runs"
DROP INDEX "billingperiodicrealizationrun_namespace_period_to";
-- reverse: create index "billingperiodicrealizationrun_namespace" to table: "billing_periodic_realization_runs"
DROP INDEX "billingperiodicrealizationrun_namespace";
-- reverse: create index "billingperiodicrealizationrun_id" to table: "billing_periodic_realization_runs"
DROP INDEX "billingperiodicrealizationrun_id";
-- reverse: create index "billing_periodic_realization_runs_previous_run_id_key" to table: "billing_periodic_realization_runs"
DROP INDEX "billing_periodic_realization_runs_previous_run_id_key";
-- reverse: create index "billing_periodic_realization_runs_one_running" to table: "billing_periodic_realization_runs"
DROP INDEX "billing_periodic_realization_runs_one_running";
-- reverse: create index "billing_periodic_realization_runs_one_first" to table: "billing_periodic_realization_runs"
DROP INDEX "billing_periodic_realization_runs_one_first";
-- reverse: create index "billing_periodic_realization_runs_due" to table: "billing_periodic_realization_runs"
DROP INDEX "billing_periodic_realization_runs_due";
-- reverse: create "billing_periodic_realization_runs" table
DROP TABLE "billing_periodic_realization_runs";
