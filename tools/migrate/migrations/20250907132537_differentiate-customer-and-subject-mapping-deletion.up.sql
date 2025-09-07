-- drop index "customersubjects_namespace_subject_key" from table: "customer_subjects"
DROP INDEX "customersubjects_namespace_subject_key";
-- modify "customer_subjects" table
ALTER TABLE "customer_subjects" ADD COLUMN "customer_deleted_at" timestamptz NULL;
-- create index "customersubjects_namespace_subject_key" to table: "customer_subjects"
CREATE UNIQUE INDEX "customersubjects_namespace_subject_key" ON "customer_subjects" ("namespace", "subject_key") WHERE ((deleted_at IS NULL) AND (customer_deleted_at IS NULL));
