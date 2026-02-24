-- reverse: modify "standard_invoice_settlements" table
ALTER TABLE "standard_invoice_settlements" DROP CONSTRAINT "standard_invoice_settlements_charges_standard_invoice_settlment", DROP CONSTRAINT "standard_invoice_settlements_billing_invoice_lines_billing_invo";
-- reverse: modify "charges" table
ALTER TABLE "charges" DROP CONSTRAINT "charges_subscriptions_charge_intents", DROP CONSTRAINT "charges_subscription_phases_charge_intents", DROP CONSTRAINT "charges_subscription_items_charge_intents", DROP CONSTRAINT "charges_customers_charge_intents";
-- reverse: modify "charge_usage_baseds" table
ALTER TABLE "charge_usage_baseds" DROP CONSTRAINT "charge_usage_baseds_charges_usage_based";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_charges_flat_fee";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP CONSTRAINT "charge_credit_purchases_charges_credit_purchase";
-- reverse: modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" DROP CONSTRAINT "billing_invoice_split_line_groups_charges_billing_split_line_gr";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_standard_invoice_settlements_standard_inv", DROP CONSTRAINT "billing_invoice_lines_charges_billing_invoice_lines";
-- reverse: create index "standardinvoicesettlement_namespace_charge_id_line_id" to table: "standard_invoice_settlements"
DROP INDEX "standardinvoicesettlement_namespace_charge_id_line_id";
-- reverse: create index "standardinvoicesettlement_namespace" to table: "standard_invoice_settlements"
DROP INDEX "standardinvoicesettlement_namespace";
-- reverse: create index "standardinvoicesettlement_id" to table: "standard_invoice_settlements"
DROP INDEX "standardinvoicesettlement_id";
-- reverse: create index "standardinvoicesettlement_annotations" to table: "standard_invoice_settlements"
DROP INDEX "standardinvoicesettlement_annotations";
-- reverse: create "standard_invoice_settlements" table
DROP TABLE "standard_invoice_settlements";
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
-- reverse: create index "chargeflatfee_namespace_id" to table: "charge_flat_fees"
DROP INDEX "chargeflatfee_namespace_id";
-- reverse: create index "chargeflatfee_namespace" to table: "charge_flat_fees"
DROP INDEX "chargeflatfee_namespace";
-- reverse: create index "chargeflatfee_id" to table: "charge_flat_fees"
DROP INDEX "chargeflatfee_id";
-- reverse: create "charge_flat_fees" table
DROP TABLE "charge_flat_fees";
-- reverse: create index "chargecreditpurchase_namespace" to table: "charge_credit_purchases"
DROP INDEX "chargecreditpurchase_namespace";
-- reverse: create index "chargecreditpurchase_id" to table: "charge_credit_purchases"
DROP INDEX "chargecreditpurchase_id";
-- reverse: create "charge_credit_purchases" table
DROP TABLE "charge_credit_purchases";
-- reverse: modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" DROP COLUMN "charge_id";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP COLUMN "charge_id", DROP COLUMN "billing_invoice_line_standard_invoice_settlments";
