-- reverse: create index "chargeusagebasedrunpayment_namespace" to table: "charge_usage_based_run_payments"
DROP INDEX "chargeusagebasedrunpayment_namespace";
-- reverse: create index "chargeusagebasedrunpayment_id" to table: "charge_usage_based_run_payments"
DROP INDEX "chargeusagebasedrunpayment_id";
-- reverse: create index "chargeusagebasedrunpayment_annotations" to table: "charge_usage_based_run_payments"
DROP INDEX "chargeusagebasedrunpayment_annotations";
-- reverse: create index "charge_usage_based_run_payments_run_id_key" to table: "charge_usage_based_run_payments"
DROP INDEX "charge_usage_based_run_payments_run_id_key";
-- reverse: create "charge_usage_based_run_payments" table
DROP TABLE "charge_usage_based_run_payments";
-- reverse: create index "chargeusagebasedruninvoicedusage_namespace" to table: "charge_usage_based_run_invoiced_usages"
DROP INDEX "chargeusagebasedruninvoicedusage_namespace";
-- reverse: create index "chargeusagebasedruninvoicedusage_id" to table: "charge_usage_based_run_invoiced_usages"
DROP INDEX "chargeusagebasedruninvoicedusage_id";
-- reverse: create index "chargeusagebasedruninvoicedusage_annotations" to table: "charge_usage_based_run_invoiced_usages"
DROP INDEX "chargeusagebasedruninvoicedusage_annotations";
-- reverse: create index "charge_usage_based_run_invoiced_usages_run_id_key" to table: "charge_usage_based_run_invoiced_usages"
DROP INDEX "charge_usage_based_run_invoiced_usages_run_id_key";
-- reverse: create "charge_usage_based_run_invoiced_usages" table
DROP TABLE "charge_usage_based_run_invoiced_usages";
-- reverse: create index "chargeusagebasedruncreditallocations_namespace" to table: "charge_usage_based_run_credit_allocations"
DROP INDEX "chargeusagebasedruncreditallocations_namespace";
-- reverse: create index "chargeusagebasedruncreditallocations_id" to table: "charge_usage_based_run_credit_allocations"
DROP INDEX "chargeusagebasedruncreditallocations_id";
-- reverse: create index "chargeusagebasedruncreditallocations_annotations" to table: "charge_usage_based_run_credit_allocations"
DROP INDEX "chargeusagebasedruncreditallocations_annotations";
-- reverse: create "charge_usage_based_run_credit_allocations" table
DROP TABLE "charge_usage_based_run_credit_allocations";
-- reverse: create index "chargeusagebasedruns_namespace_charge_id" to table: "charge_usage_based_runs"
DROP INDEX "chargeusagebasedruns_namespace_charge_id";
-- reverse: create index "chargeusagebasedruns_namespace" to table: "charge_usage_based_runs"
DROP INDEX "chargeusagebasedruns_namespace";
-- reverse: create index "chargeusagebasedruns_id" to table: "charge_usage_based_runs"
DROP INDEX "chargeusagebasedruns_id";
-- reverse: create "charge_usage_based_runs" table
DROP TABLE "charge_usage_based_runs";
-- reverse: create index "chargeusagebased_namespace_id" to table: "charge_usage_based"
DROP INDEX "chargeusagebased_namespace_id";
-- reverse: create index "chargeusagebased_namespace" to table: "charge_usage_based"
DROP INDEX "chargeusagebased_namespace";
-- reverse: create index "chargeusagebased_id" to table: "charge_usage_based"
DROP INDEX "chargeusagebased_id";
-- reverse: create "charge_usage_based" table
DROP TABLE "charge_usage_based";
