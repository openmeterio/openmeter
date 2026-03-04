-- modify "charge_standard_invoice_payment_settlements" table
ALTER TABLE "charge_standard_invoice_payment_settlements" DROP COLUMN "charge_id";
-- create index "chargestandardinvoicepaymentsettlement_namespace_line_id" to table: "charge_standard_invoice_payment_settlements"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "chargestandardinvoicepaymentsettlement_namespace_line_id" ON "charge_standard_invoice_payment_settlements" ("namespace", "line_id") WHERE ((line_id IS NOT NULL) AND (deleted_at IS NULL));
-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD COLUMN "std_invoice_payment_settlement_id" character(26) NULL, ADD CONSTRAINT "charge_flat_fees_charge_standard_invoice_payment_settlements_ch" FOREIGN KEY ("std_invoice_payment_settlement_id") REFERENCES "charge_standard_invoice_payment_settlements" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "charge_flat_fees_std_invoice_payment_settlement_id_key" to table: "charge_flat_fees"
CREATE UNIQUE INDEX "charge_flat_fees_std_invoice_payment_settlement_id_key" ON "charge_flat_fees" ("std_invoice_payment_settlement_id");
