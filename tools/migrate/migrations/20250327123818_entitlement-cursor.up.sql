-- create index "entitlement_created_at_id" to table: "entitlements"
-- atlas:nolint
CREATE UNIQUE INDEX "entitlement_created_at_id" ON "entitlements" ("created_at", "id");
