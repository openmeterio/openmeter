-- reverse: create index "chargeusagebaseddetailedline_tax_code_id" to table: "charge_usage_based_detailed_line"
DROP INDEX "chargeusagebaseddetailedline_tax_code_id";
-- reverse: create index "chargeusagebaseddetailedline_namespace_run_id" to table: "charge_usage_based_detailed_line"
DROP INDEX "chargeusagebaseddetailedline_namespace_run_id";
-- reverse: create index "chargeusagebaseddetailedline_namespace_id" to table: "charge_usage_based_detailed_line"
DROP INDEX "chargeusagebaseddetailedline_namespace_id";
-- reverse: create index "chargeusagebaseddetailedline_namespace_charge_id" to table: "charge_usage_based_detailed_line"
DROP INDEX "chargeusagebaseddetailedline_namespace_charge_id";
-- reverse: create index "chargeusagebaseddetailedline_namespace" to table: "charge_usage_based_detailed_line"
DROP INDEX "chargeusagebaseddetailedline_namespace";
-- reverse: create index "chargeusagebaseddetailedline_id" to table: "charge_usage_based_detailed_line"
DROP INDEX "chargeusagebaseddetailedline_id";
-- reverse: create index "chargeusagebaseddetailedline_annotations" to table: "charge_usage_based_detailed_line"
DROP INDEX "chargeusagebaseddetailedline_annotations";
-- reverse: create index "chargeubdetailedline_ns_charge_run_child_id" to table: "charge_usage_based_detailed_line"
DROP INDEX "chargeubdetailedline_ns_charge_run_child_id";
-- reverse: create "charge_usage_based_detailed_line" table
DROP TABLE "charge_usage_based_detailed_line";
-- reverse: create index "chargeflatfeedetailedline_tax_code_id" to table: "charge_flat_fee_detailed_line"
DROP INDEX "chargeflatfeedetailedline_tax_code_id";
-- reverse: create index "chargeflatfeedetailedline_namespace_id" to table: "charge_flat_fee_detailed_line"
DROP INDEX "chargeflatfeedetailedline_namespace_id";
-- reverse: create index "chargeflatfeedetailedline_namespace_charge_id" to table: "charge_flat_fee_detailed_line"
DROP INDEX "chargeflatfeedetailedline_namespace_charge_id";
-- reverse: create index "chargeflatfeedetailedline_namespace" to table: "charge_flat_fee_detailed_line"
DROP INDEX "chargeflatfeedetailedline_namespace";
-- reverse: create index "chargeflatfeedetailedline_id" to table: "charge_flat_fee_detailed_line"
DROP INDEX "chargeflatfeedetailedline_id";
-- reverse: create index "chargeflatfeedetailedline_annotations" to table: "charge_flat_fee_detailed_line"
DROP INDEX "chargeflatfeedetailedline_annotations";
-- reverse: create index "chargeffdetailedline_ns_charge_child_id" to table: "charge_flat_fee_detailed_line"
DROP INDEX "chargeffdetailedline_ns_charge_child_id";
-- reverse: create "charge_flat_fee_detailed_line" table
DROP TABLE "charge_flat_fee_detailed_line";
