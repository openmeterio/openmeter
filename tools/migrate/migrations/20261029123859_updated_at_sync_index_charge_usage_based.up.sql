-- create index "chargeusagebased_namespace_updated_at" to table: "charge_usage_based"
CREATE INDEX "chargeusagebased_namespace_updated_at" ON "charge_usage_based" ("namespace", "updated_at");
