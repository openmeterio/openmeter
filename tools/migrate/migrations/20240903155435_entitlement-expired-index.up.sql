-- create index "entitlement_current_usage_period_end_deleted_at" to table: "entitlements"
CREATE INDEX "entitlement_current_usage_period_end_deleted_at" ON "entitlements" ("current_usage_period_end", "deleted_at");
