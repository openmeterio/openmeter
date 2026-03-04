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
