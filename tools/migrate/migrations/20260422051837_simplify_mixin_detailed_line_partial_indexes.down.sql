-- reverse: create index "chargeubdetailedline_ns_charge_run_child_id" to table: "charge_usage_based_run_detailed_line"
DROP INDEX "chargeubdetailedline_ns_charge_run_child_id";
-- reverse: drop index "chargeubdetailedline_ns_charge_run_child_id" from table: "charge_usage_based_run_detailed_line"
CREATE UNIQUE INDEX "chargeubdetailedline_ns_charge_run_child_id" ON "charge_usage_based_run_detailed_line" ("namespace", "charge_id", "run_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- reverse: create index "chargeffdetailedline_ns_charge_child_id" to table: "charge_flat_fee_detailed_line"
DROP INDEX "chargeffdetailedline_ns_charge_child_id";
-- reverse: drop index "chargeffdetailedline_ns_charge_child_id" from table: "charge_flat_fee_detailed_line"
CREATE UNIQUE INDEX "chargeffdetailedline_ns_charge_child_id" ON "charge_flat_fee_detailed_line" ("namespace", "charge_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- reverse: create index "billingstdinvdetailedline_ns_parent_child_id" to table: "billing_standard_invoice_detailed_lines"
DROP INDEX "billingstdinvdetailedline_ns_parent_child_id";
-- reverse: drop index "billingstdinvdetailedline_ns_parent_child_id" from table: "billing_standard_invoice_detailed_lines"
CREATE UNIQUE INDEX "billingstdinvdetailedline_ns_parent_child_id" ON "billing_standard_invoice_detailed_lines" ("namespace", "parent_line_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
