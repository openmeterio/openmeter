-- create index "chargecreditpurchase_namespace_updated_at" to table: "charge_credit_purchases"
CREATE INDEX "chargecreditpurchase_namespace_updated_at" ON "charge_credit_purchases" ("namespace", "updated_at");
