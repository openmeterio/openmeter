-- reverse: modify "charges" table
ALTER TABLE "charges" DROP CONSTRAINT "charges_subscriptions_charge_intents", DROP CONSTRAINT "charges_subscription_phases_charge_intents", DROP CONSTRAINT "charges_subscription_items_charge_intents", DROP CONSTRAINT "charges_customers_charge_intents";
-- reverse: modify "charge_usage_baseds" table
ALTER TABLE "charge_usage_baseds" DROP CONSTRAINT "charge_usage_baseds_charges_usage_based";
-- reverse: modify "charge_standard_invoice_realizations" table
ALTER TABLE "charge_standard_invoice_realizations" DROP CONSTRAINT "charge_standard_invoice_realizations_charges_standard_invoice_r", DROP CONSTRAINT "charge_standard_invoice_realizations_billing_invoice_lines_bill";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_charges_flat_fee";
-- reverse: modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" DROP CONSTRAINT "billing_invoice_split_line_groups_charges_billing_split_line_gr";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_charges_billing_invoice_lines", DROP CONSTRAINT "billing_invoice_lines_charge_standard_invoice_realizations_stan";
-- reverse: create index "charge_namespace_id" to table: "charges"
DROP INDEX "charge_namespace_id";
-- reverse: create index "charge_namespace_customer_id_unique_reference_id" to table: "charges"
DROP INDEX "charge_namespace_customer_id_unique_reference_id";
-- reverse: create index "charge_namespace" to table: "charges"
DROP INDEX "charge_namespace";
-- reverse: create index "charge_id" to table: "charges"
DROP INDEX "charge_id";
-- reverse: create index "charge_annotations" to table: "charges"
DROP INDEX "charge_annotations";
-- reverse: create "charges" table
DROP TABLE "charges";
-- reverse: create index "chargeusagebased_namespace_id" to table: "charge_usage_baseds"
DROP INDEX "chargeusagebased_namespace_id";
-- reverse: create index "chargeusagebased_namespace" to table: "charge_usage_baseds"
DROP INDEX "chargeusagebased_namespace";
-- reverse: create index "chargeusagebased_id" to table: "charge_usage_baseds"
DROP INDEX "chargeusagebased_id";
-- reverse: create "charge_usage_baseds" table
DROP TABLE "charge_usage_baseds";
-- reverse: create index "chargestandardinvoicerealization_namespace_charge_id_line_id" to table: "charge_standard_invoice_realizations"
DROP INDEX "chargestandardinvoicerealization_namespace_charge_id_line_id";
-- reverse: create index "chargestandardinvoicerealization_namespace" to table: "charge_standard_invoice_realizations"
DROP INDEX "chargestandardinvoicerealization_namespace";
-- reverse: create index "chargestandardinvoicerealization_id" to table: "charge_standard_invoice_realizations"
DROP INDEX "chargestandardinvoicerealization_id";
-- reverse: create index "chargestandardinvoicerealization_annotations" to table: "charge_standard_invoice_realizations"
DROP INDEX "chargestandardinvoicerealization_annotations";
-- reverse: create "charge_standard_invoice_realizations" table
DROP TABLE "charge_standard_invoice_realizations";
-- reverse: create index "chargeflatfee_namespace_id" to table: "charge_flat_fees"
DROP INDEX "chargeflatfee_namespace_id";
-- reverse: create index "chargeflatfee_namespace" to table: "charge_flat_fees"
DROP INDEX "chargeflatfee_namespace";
-- reverse: create index "chargeflatfee_id" to table: "charge_flat_fees"
DROP INDEX "chargeflatfee_id";
-- reverse: create "charge_flat_fees" table
DROP TABLE "charge_flat_fees";
-- reverse: modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" DROP COLUMN "charge_id";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP COLUMN "charge_id", DROP COLUMN "billing_invoice_line_standard_invoice_realizations";
