-- reverse: create index "charge_flat_fees_std_invoice_payment_settlement_id_key" to table: "charge_flat_fees"
DROP INDEX "charge_flat_fees_std_invoice_payment_settlement_id_key";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_charge_standard_invoice_payment_settlements_ch", DROP COLUMN "std_invoice_payment_settlement_id";
-- reverse: create index "chargestandardinvoicepaymentsettlement_namespace_line_id" to table: "charge_standard_invoice_payment_settlements"
DROP INDEX "chargestandardinvoicepaymentsettlement_namespace_line_id";
-- reverse: modify "charge_standard_invoice_payment_settlements" table
ALTER TABLE "charge_standard_invoice_payment_settlements" ADD COLUMN "charge_id" character(26) NOT NULL;

CREATE UNIQUE INDEX "chargestandardinvoicepaymentsettlement_namespace_charge_id_line" ON "charge_standard_invoice_payment_settlements" ("namespace", "charge_id", "line_id") WHERE ((line_id IS NOT NULL) AND (deleted_at IS NULL));
CREATE UNIQUE INDEX "charge_standard_invoice_payment_settlements_charge_id_key" ON "charge_standard_invoice_payment_settlements" ("charge_id");
