-- reverse: create index "chargecreditpurchaseinvoicedpayment_namespace_charge_id" to table: "charge_credit_purchase_invoiced_payments"
DROP INDEX "chargecreditpurchaseinvoicedpayment_namespace_charge_id";
-- reverse: create index "chargecreditpurchaseinvoicedpayment_namespace" to table: "charge_credit_purchase_invoiced_payments"
DROP INDEX "chargecreditpurchaseinvoicedpayment_namespace";
-- reverse: create index "chargecreditpurchaseinvoicedpayment_id" to table: "charge_credit_purchase_invoiced_payments"
DROP INDEX "chargecreditpurchaseinvoicedpayment_id";
-- reverse: create index "chargecreditpurchaseinvoicedpayment_annotations" to table: "charge_credit_purchase_invoiced_payments"
DROP INDEX "chargecreditpurchaseinvoicedpayment_annotations";
-- reverse: create index "charge_credit_purchase_invoiced_payments_line_id_key" to table: "charge_credit_purchase_invoiced_payments"
DROP INDEX "charge_credit_purchase_invoiced_payments_line_id_key";
-- reverse: create index "charge_credit_purchase_invoiced_payments_charge_id_key" to table: "charge_credit_purchase_invoiced_payments"
DROP INDEX "charge_credit_purchase_invoiced_payments_charge_id_key";
-- reverse: create "charge_credit_purchase_invoiced_payments" table
DROP TABLE "charge_credit_purchase_invoiced_payments";
-- reverse: modify "charge_usage_based_run_payments" table
ALTER TABLE "charge_usage_based_run_payments" DROP COLUMN "invoice_id";
-- reverse: modify "charge_flat_fee_payments" table
ALTER TABLE "charge_flat_fee_payments" DROP COLUMN "invoice_id";
