-- reverse: create index "customersubjects_namespace_subject_key_deleted_at" to table: "customer_subjects"
DROP INDEX "customersubjects_namespace_subject_key_deleted_at";
-- reverse: create index "customersubjects_namespace_customer_id" to table: "customer_subjects"
DROP INDEX "customersubjects_namespace_customer_id";
-- reverse: modify "customer_subjects" table
ALTER TABLE "customer_subjects" ADD COLUMN "is_deleted" boolean NOT NULL DEFAULT false;
