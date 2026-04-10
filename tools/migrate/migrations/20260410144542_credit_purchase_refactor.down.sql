-- reverse: create index "chargecreditpurchasecreditgrant_namespace_charge_id" to table: "charge_credit_purchase_credit_grants"
DROP INDEX "chargecreditpurchasecreditgrant_namespace_charge_id";
-- reverse: create index "chargecreditpurchasecreditgrant_namespace" to table: "charge_credit_purchase_credit_grants"
DROP INDEX "chargecreditpurchasecreditgrant_namespace";
-- reverse: create index "chargecreditpurchasecreditgrant_id" to table: "charge_credit_purchase_credit_grants"
DROP INDEX "chargecreditpurchasecreditgrant_id";
-- reverse: create index "charge_credit_purchase_credit_grants_charge_id_key" to table: "charge_credit_purchase_credit_grants"
DROP INDEX "charge_credit_purchase_credit_grants_charge_id_key";
-- reverse: create "charge_credit_purchase_credit_grants" table
DROP TABLE "charge_credit_purchase_credit_grants";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP COLUMN "status_detailed", ADD COLUMN "credit_granted_at" timestamptz NULL, ADD COLUMN "credit_grant_transaction_group_id" character(26) NULL;
