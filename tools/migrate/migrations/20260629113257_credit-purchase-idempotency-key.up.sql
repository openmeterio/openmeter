-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ADD COLUMN "key" character varying NULL;
-- create index "chargecreditpurchase_namespace_key" to table: "charge_credit_purchases"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "chargecreditpurchase_namespace_key" ON "charge_credit_purchases" ("namespace", "key") WHERE ((key IS NOT NULL) AND (deleted_at IS NULL));
