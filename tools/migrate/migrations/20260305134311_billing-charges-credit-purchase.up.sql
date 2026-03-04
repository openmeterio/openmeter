-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP COLUMN "status", ADD COLUMN "credit_grant_transaction_group_id" character(26) NULL, ADD COLUMN "credit_granted_at" timestamptz NULL;
-- create "charge_external_payment_settlements" table
CREATE TABLE "charge_external_payment_settlements" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "status" character varying NOT NULL,
  "amount" numeric NOT NULL,
  "authorized_transaction_group_id" character(26) NULL,
  "authorized_at" timestamptz NULL,
  "settled_transaction_group_id" character(26) NULL,
  "settled_at" timestamptz NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_external_payment_settlements_charge_credit_purchases_cha" FOREIGN KEY ("charge_id") REFERENCES "charge_credit_purchases" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "charge_external_payment_settlements_charge_id_key" to table: "charge_external_payment_settlements"
CREATE UNIQUE INDEX "charge_external_payment_settlements_charge_id_key" ON "charge_external_payment_settlements" ("charge_id");
-- create index "chargeexternalpaymentsettlement_annotations" to table: "charge_external_payment_settlements"
CREATE INDEX "chargeexternalpaymentsettlement_annotations" ON "charge_external_payment_settlements" USING gin ("annotations");
-- create index "chargeexternalpaymentsettlement_charge_id" to table: "charge_external_payment_settlements"
CREATE INDEX "chargeexternalpaymentsettlement_charge_id" ON "charge_external_payment_settlements" ("charge_id");
-- create index "chargeexternalpaymentsettlement_id" to table: "charge_external_payment_settlements"
CREATE UNIQUE INDEX "chargeexternalpaymentsettlement_id" ON "charge_external_payment_settlements" ("id");
-- create index "chargeexternalpaymentsettlement_namespace" to table: "charge_external_payment_settlements"
CREATE INDEX "chargeexternalpaymentsettlement_namespace" ON "charge_external_payment_settlements" ("namespace");
-- create index "chargeexternalpaymentsettlement_namespace_charge_id" to table: "charge_external_payment_settlements"
CREATE INDEX "chargeexternalpaymentsettlement_namespace_charge_id" ON "charge_external_payment_settlements" ("namespace", "charge_id");
