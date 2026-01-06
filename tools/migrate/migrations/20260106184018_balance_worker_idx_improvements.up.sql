-- create index "customer_namespace_key_deleted_at" to table: "customers"
CREATE INDEX "customer_namespace_key_deleted_at" ON "customers" ("namespace", "key", "deleted_at");
-- create index "feature_namespace_meter_slug" to table: "features"
CREATE INDEX "feature_namespace_meter_slug" ON "features" ("namespace", "meter_slug") WHERE (archived_at IS NULL);
