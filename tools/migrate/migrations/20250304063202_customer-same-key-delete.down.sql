-- reverse: create index "customer_namespace_key" to table: "customers"
DROP INDEX "customer_namespace_key";
-- reverse: modify "customers" table
ALTER TABLE "customers" ADD COLUMN "is_deleted" boolean NOT NULL DEFAULT false;
-- reverse: create index "customersubjects_namespace_subject_key" to table: "customer_subjects"
DROP INDEX "customersubjects_namespace_subject_key";
-- reverse: create index "customersubjects_namespace_customer_id_deleted_at" to table: "customer_subjects"
DROP INDEX "customersubjects_namespace_customer_id_deleted_at";
-- reverse: modify "customer_subjects" table
ALTER TABLE "customer_subjects" ADD COLUMN "is_deleted" boolean NOT NULL DEFAULT false;
