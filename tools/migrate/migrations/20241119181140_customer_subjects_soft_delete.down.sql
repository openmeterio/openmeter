-- reverse: create index "customersubjects_namespace_subject_key_is_deleted" to table: "customer_subjects"
DROP INDEX "customersubjects_namespace_subject_key_is_deleted";
-- reverse: create index "customersubjects_namespace_customer_id_is_deleted" to table: "customer_subjects"
DROP INDEX "customersubjects_namespace_customer_id_is_deleted";
-- reverse: modify "customer_subjects" table
ALTER TABLE "customer_subjects" DROP COLUMN "is_deleted", DROP COLUMN "deleted_at";
-- atlas:nolint MF101
-- reverse: drop index "customersubjects_namespace_subject_key" from table: "customer_subjects"
CREATE UNIQUE INDEX "customersubjects_namespace_subject_key" ON "customer_subjects" ("namespace", "subject_key");
-- atlas:nolint MF101
-- reverse: drop index "customersubjects_customer_id_subject_key" from table: "customer_subjects"
CREATE UNIQUE INDEX "customersubjects_customer_id_subject_key" ON "customer_subjects" ("customer_id", "subject_key");
