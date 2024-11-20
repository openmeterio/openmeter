-- modify "customer_subjects" table
ALTER TABLE "customer_subjects" DROP COLUMN "is_deleted";
-- create index "customersubjects_namespace_customer_id" to table: "customer_subjects"
CREATE INDEX "customersubjects_namespace_customer_id" ON "customer_subjects" ("namespace", "customer_id");
-- create index "customersubjects_namespace_subject_key_deleted_at" to table: "customer_subjects"
CREATE UNIQUE INDEX "customersubjects_namespace_subject_key_deleted_at" ON "customer_subjects" ("namespace", "subject_key", "deleted_at") WHERE (deleted_at IS NULL);
