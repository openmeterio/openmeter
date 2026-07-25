-- reverse: create index "chargecreditpurchase_namespace_key" to table: "charge_credit_purchases"
DROP INDEX "chargecreditpurchase_namespace_key";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP COLUMN "key";
