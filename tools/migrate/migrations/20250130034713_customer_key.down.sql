-- reverse: create index "customer_namespace_key_is_deleted" to table: "customers"
DROP INDEX "customer_namespace_key_is_deleted";
-- reverse: create index "customer_is_deleted" to table: "customers"
DROP INDEX "customer_is_deleted";
-- reverse: modify "customers" table
ALTER TABLE "customers" DROP COLUMN "is_deleted", DROP COLUMN "key";
-- reverse: drop index "customer_deleted_at" from table: "customers"
CREATE INDEX "customer_deleted_at" ON "customers" ("deleted_at");
