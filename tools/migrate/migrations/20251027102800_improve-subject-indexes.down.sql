-- reverse: create index "subject_namespace_key_deleted_at_unique" to table: "subjects"
DROP INDEX "subject_namespace_key_deleted_at_unique";
-- reverse: create index "subject_namespace_key_deleted_at" to table: "subjects"
DROP INDEX "subject_namespace_key_deleted_at";
-- reverse: drop index "subject_namespace_key_deleted_at" from table: "subjects"
CREATE UNIQUE INDEX "subject_namespace_key_deleted_at" ON "subjects" ("namespace", "key", "deleted_at") WHERE (deleted_at IS NULL);
