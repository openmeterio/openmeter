-- reverse: create index "chargecreditpurchase_namespace_customer_id_key" to table: "charge_credit_purchases"
DROP INDEX "chargecreditpurchase_namespace_customer_id_key";
-- reverse: drop index "chargecreditpurchase_namespace_key" from table: "charge_credit_purchases"
CREATE UNIQUE INDEX "chargecreditpurchase_namespace_key" ON "charge_credit_purchases" ("namespace", "key") WHERE ((key IS NOT NULL) AND (deleted_at IS NULL));
