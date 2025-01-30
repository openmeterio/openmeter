-- modify "customers" table
ALTER TABLE "customers" ADD COLUMN "key" character varying NULL, ADD COLUMN "is_deleted" boolean NOT NULL DEFAULT false;
-- create index "customer_is_deleted" to table: "customers"
CREATE INDEX "customer_is_deleted" ON "customers" ("is_deleted");
-- create index "customer_namespace_key_is_deleted" to table: "customers"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "customer_namespace_key_is_deleted" ON "customers" ("namespace", "key", "is_deleted");
-- data migration
UPDATE "customers" SET "is_deleted" = true WHERE "deleted_at" IS NOT NULL;
-- drop index "customer_deleted_at" from table: "customers"
DROP INDEX "customer_deleted_at";
