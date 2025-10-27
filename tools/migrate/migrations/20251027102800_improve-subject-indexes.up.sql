-- rename index "subject_namespace_key_deleted_at" to "subject_namespace_key_deleted_at_unique"
ALTER INDEX "subject_namespace_key_deleted_at" RENAME TO "subject_namespace_key_deleted_at_unique";
-- create index "subject_namespace_key_deleted_at" to table: "subjects"
CREATE INDEX "subject_namespace_key_deleted_at" ON "subjects" ("namespace", "key", "deleted_at");
