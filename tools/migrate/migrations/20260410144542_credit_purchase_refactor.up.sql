-- modify "charge_credit_purchases" table
-- atlas:nolint DS103
ALTER TABLE "charge_credit_purchases" DROP COLUMN "credit_grant_transaction_group_id", DROP COLUMN "credit_granted_at", ADD COLUMN "status_detailed" character varying NULL;
UPDATE "charge_credit_purchases" SET "status_detailed" = "status";
-- atlas:nolint MF104
ALTER TABLE "charge_credit_purchases" ALTER COLUMN "status_detailed" SET NOT NULL;
-- create "charge_credit_purchase_credit_grants" table
CREATE TABLE "charge_credit_purchase_credit_grants" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "transaction_group_id" character(26) NOT NULL,
  "granted_at" timestamptz NOT NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_credit_purchase_credit_grants_charge_credit_purchases_cr" FOREIGN KEY ("charge_id") REFERENCES "charge_credit_purchases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "charge_credit_purchase_credit_grants_charge_id_key" to table: "charge_credit_purchase_credit_grants"
CREATE UNIQUE INDEX "charge_credit_purchase_credit_grants_charge_id_key" ON "charge_credit_purchase_credit_grants" ("charge_id");
-- create index "chargecreditpurchasecreditgrant_id" to table: "charge_credit_purchase_credit_grants"
CREATE UNIQUE INDEX "chargecreditpurchasecreditgrant_id" ON "charge_credit_purchase_credit_grants" ("id");
-- create index "chargecreditpurchasecreditgrant_namespace" to table: "charge_credit_purchase_credit_grants"
CREATE INDEX "chargecreditpurchasecreditgrant_namespace" ON "charge_credit_purchase_credit_grants" ("namespace");
-- create index "chargecreditpurchasecreditgrant_namespace_charge_id" to table: "charge_credit_purchase_credit_grants"
CREATE UNIQUE INDEX "chargecreditpurchasecreditgrant_namespace_charge_id" ON "charge_credit_purchase_credit_grants" ("namespace", "charge_id");
