-- create index "chargecreditpurchase_namespace_customer_id" to table: "charge_credit_purchases"
CREATE INDEX CONCURRENTLY "chargecreditpurchase_namespace_customer_id" ON "charge_credit_purchases" ("namespace", "customer_id") WHERE ((deleted_at IS NULL) AND ((status)::text <> 'final'::text));
