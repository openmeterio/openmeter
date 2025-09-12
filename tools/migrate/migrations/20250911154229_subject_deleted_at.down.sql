-- reverse: create index "subject_namespace_key" to table: "subjects"
DROP INDEX "subject_namespace_key";
-- reverse: modify "subjects" table
ALTER TABLE "subjects" DROP COLUMN "deleted_at";
-- reverse: drop index "subject_key_namespace" from table: "subjects"
CREATE UNIQUE INDEX "subject_key_namespace" ON "subjects" ("key", "namespace");
