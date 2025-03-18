-- create index "entitlement_created_at_id" to table: "entitlements"
CREATE UNIQUE INDEX "entitlement_created_at_id" ON "entitlements" ("created_at", "id");
