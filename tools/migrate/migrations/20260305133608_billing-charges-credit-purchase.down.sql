-- reverse: create index "chargeexternalpaymentsettlement_namespace_charge_id" to table: "charge_external_payment_settlements"
DROP INDEX "chargeexternalpaymentsettlement_namespace_charge_id";
-- reverse: create index "chargeexternalpaymentsettlement_namespace" to table: "charge_external_payment_settlements"
DROP INDEX "chargeexternalpaymentsettlement_namespace";
-- reverse: create index "chargeexternalpaymentsettlement_id" to table: "charge_external_payment_settlements"
DROP INDEX "chargeexternalpaymentsettlement_id";
-- reverse: create index "chargeexternalpaymentsettlement_charge_id" to table: "charge_external_payment_settlements"
DROP INDEX "chargeexternalpaymentsettlement_charge_id";
-- reverse: create index "chargeexternalpaymentsettlement_annotations" to table: "charge_external_payment_settlements"
DROP INDEX "chargeexternalpaymentsettlement_annotations";
-- reverse: create index "charge_external_payment_settlements_charge_id_key" to table: "charge_external_payment_settlements"
DROP INDEX "charge_external_payment_settlements_charge_id_key";
-- reverse: create "charge_external_payment_settlements" table
DROP TABLE "charge_external_payment_settlements";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP COLUMN "credit_granted_at", DROP COLUMN "credit_grant_transaction_group_id", ADD COLUMN "status" character varying NOT NULL;
