-- create index "chargecreditpurchase_namespace_subscription_id" to table: "charge_credit_purchases"
CREATE INDEX CONCURRENTLY "chargecreditpurchase_namespace_subscription_id" ON "charge_credit_purchases" ("namespace", "subscription_id");
