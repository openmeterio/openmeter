-- reverse: modify "customers" table
ALTER TABLE "customers" ADD COLUMN "is_deleted" boolean;
UPDATE "customers" SET "is_deleted" = deleted_at IS NOT NULL;
ALTER TABLE "customers" ALTER COLUMN "is_deleted" SET NOT NULL;
CREATE UNIQUE INDEX "customer_namespace_key_is_deleted" ON "customers" ("namespace", "key", "is_deleted");
CREATE INDEX "customer_is_deleted" ON "customers" ("is_deleted");
-- reverse: create index "customersubjects_namespace_subject_key" to table: "customer_subjects"
DROP INDEX "customersubjects_namespace_subject_key";
-- reverse: create index "customersubjects_namespace_customer_id_deleted_at" to table: "customer_subjects"
DROP INDEX "customersubjects_namespace_customer_id_deleted_at";
-- reverse: modify "customer_subjects" table
ALTER TABLE "customer_subjects" ADD COLUMN "is_deleted" boolean NOT NULL DEFAULT false;
CREATE INDEX "customersubjects_namespace_customer_id_is_deleted" ON "customer_subjects" ("namespace", "customer_id", "is_deleted");
-- atlas:nolint MF101
-- create index "customersubjects_namespace_subject_key_is_deleted" to table: "customer_subjects"
CREATE UNIQUE INDEX "customersubjects_namespace_subject_key_is_deleted" ON "customer_subjects" ("namespace", "subject_key", "is_deleted") WHERE (is_deleted = false);

