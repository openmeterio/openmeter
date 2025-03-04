-- modify "customer_subjects" table
DROP INDEX "customersubjects_namespace_customer_id_is_deleted";
DROP INDEX "customersubjects_namespace_subject_key_is_deleted";
-- atlas:nolint DS103
ALTER TABLE "customer_subjects" DROP COLUMN "is_deleted";
-- create index "customersubjects_namespace_customer_id_deleted_at" to table: "customer_subjects"
CREATE INDEX "customersubjects_namespace_customer_id_deleted_at" ON "customer_subjects" ("namespace", "customer_id", "deleted_at");
-- create index "customersubjects_namespace_subject_key" to table: "customer_subjects"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "customersubjects_namespace_subject_key" ON "customer_subjects" ("namespace", "subject_key") WHERE (deleted_at IS NOT NULL);
-- modify "customers" table
DROP INDEX "customer_namespace_key_is_deleted";
-- atlas:nolint DS103
ALTER TABLE "customers" DROP COLUMN "is_deleted";
-- create index "customer_namespace_key" to table: "customers"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "customer_namespace_key" ON "customers" ("namespace", "key") WHERE (deleted_at IS NULL);
