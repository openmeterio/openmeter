-- reverse: create index "entitlement_namespace_subject_id" to table: "entitlements"
DROP INDEX "entitlement_namespace_subject_id";
-- reverse: create index "entitlement_namespace_id_customer_id" to table: "entitlements"
DROP INDEX "entitlement_namespace_id_customer_id";
-- reverse: create index "entitlement_namespace_customer_id" to table: "entitlements"
DROP INDEX "entitlement_namespace_customer_id";
-- reverse: modify "entitlements" table
ALTER TABLE "entitlements" DROP CONSTRAINT "entitlements_customers_entitlements", DROP COLUMN "customer_id";
-- reverse: drop index "entitlement_namespace_id_subject_key" from table: "entitlements"
CREATE INDEX "entitlement_namespace_id_subject_key" ON "entitlements" ("namespace", "id", "subject_key");
