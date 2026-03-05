-- reverse: create index "charge_flat_fees_std_invoice_payment_settlement_id_key" to table: "charge_flat_fees"
DROP INDEX "charge_flat_fees_std_invoice_payment_settlement_id_key";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_charge_standard_invoice_payment_settlements_ch", DROP COLUMN "std_invoice_payment_settlement_id";
-- reverse: create index "chargestandardinvoicepaymentsettlement_namespace_line_id" to table: "charge_standard_invoice_payment_settlements"
DROP INDEX "chargestandardinvoicepaymentsettlement_namespace_line_id";
-- reverse: modify "charge_standard_invoice_payment_settlements" table
ALTER TABLE "charge_standard_invoice_payment_settlements" ADD COLUMN "charge_id" character(26) NOT NULL, ADD CONSTRAINT "charge_standard_invoice_payment_settlements_charge_flat_fees_ch" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- recreate indexes on charge_id that were implicitly dropped when the column was dropped
CREATE UNIQUE INDEX "charge_standard_invoice_payment_settlements_charge_id_key" ON "charge_standard_invoice_payment_settlements" ("charge_id");
CREATE UNIQUE INDEX "chargestandardinvoicepaymentsettlement_namespace_charge_id_line" ON "charge_standard_invoice_payment_settlements" ("namespace", "charge_id", "line_id") WHERE ((line_id IS NOT NULL) AND (deleted_at IS NULL));
-- reverse: create index "charge_credit_purchases_external_payment_settlement_id_key" to table: "charge_credit_purchases"
DROP INDEX "charge_credit_purchases_external_payment_settlement_id_key";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP CONSTRAINT "charge_credit_purchases_charge_external_payment_settlements_cha", DROP COLUMN "external_payment_settlement_id", DROP COLUMN "credit_granted_at", DROP COLUMN "credit_grant_transaction_group_id", ADD COLUMN "status" character varying NOT NULL;
-- reverse: create index "chargeexternalpaymentsettlement_namespace_id" to table: "charge_external_payment_settlements"
DROP INDEX "chargeexternalpaymentsettlement_namespace_id";
-- reverse: create index "chargeexternalpaymentsettlement_namespace" to table: "charge_external_payment_settlements"
DROP INDEX "chargeexternalpaymentsettlement_namespace";
-- reverse: create index "chargeexternalpaymentsettlement_id" to table: "charge_external_payment_settlements"
DROP INDEX "chargeexternalpaymentsettlement_id";
-- reverse: create index "chargeexternalpaymentsettlement_annotations" to table: "charge_external_payment_settlements"
DROP INDEX "chargeexternalpaymentsettlement_annotations";
-- reverse: create "charge_external_payment_settlements" table
DROP TABLE "charge_external_payment_settlements";
