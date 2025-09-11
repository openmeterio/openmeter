-- drop index "subject_key_namespace" from table: "subjects"
DROP INDEX "subject_key_namespace";
-- modify "subjects" table
ALTER TABLE "subjects" ADD COLUMN "deleted_at" timestamptz NULL;
-- create index "subject_namespace_key" to table: "subjects"
CREATE UNIQUE INDEX "subject_namespace_key" ON "subjects" ("namespace", "key") WHERE (deleted_at IS NULL);
