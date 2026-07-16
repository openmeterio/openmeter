-- drop index "chargecreditpurchase_namespace_key" from table: "charge_credit_purchases"
DROP INDEX "chargecreditpurchase_namespace_key";
-- create index "chargecreditpurchase_namespace_customer_id_key" to table: "charge_credit_purchases"
-- the old (namespace, key) unique index guarantees no (namespace, customer_id, key) duplicates exist,
-- so widening the key scope cannot fail on existing data
-- atlas:nolint MF101
CREATE UNIQUE INDEX "chargecreditpurchase_namespace_customer_id_key" ON "charge_credit_purchases" ("namespace", "customer_id", "key") WHERE ((key IS NOT NULL) AND (deleted_at IS NULL));
