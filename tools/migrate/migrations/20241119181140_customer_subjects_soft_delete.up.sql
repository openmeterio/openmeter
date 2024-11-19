-- drop index "customersubjects_customer_id_subject_key" from table: "customer_subjects"
DROP INDEX "customersubjects_customer_id_subject_key";
-- drop index "customersubjects_namespace_subject_key" from table: "customer_subjects"
DROP INDEX "customersubjects_namespace_subject_key";
-- modify "customer_subjects" table
ALTER TABLE "customer_subjects" ADD COLUMN "deleted_at" timestamptz NULL, ADD COLUMN "is_deleted" boolean NOT NULL DEFAULT false;
-- create index "customersubjects_namespace_customer_id_is_deleted" to table: "customer_subjects"
CREATE INDEX "customersubjects_namespace_customer_id_is_deleted" ON "customer_subjects" ("namespace", "customer_id", "is_deleted");
-- create index "customersubjects_namespace_subject_key_is_deleted" to table: "customer_subjects"
CREATE UNIQUE INDEX "customersubjects_namespace_subject_key_is_deleted" ON "customer_subjects" ("namespace", "subject_key", "is_deleted") WHERE (is_deleted = false);
