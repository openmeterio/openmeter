-- create index "subject_namespace_id" to table: "subjects"
CREATE UNIQUE INDEX "subject_namespace_id" ON "subjects" ("namespace", "id");
-- create index "subject_namespace_key_deleted_at" to table: "subjects"
CREATE UNIQUE INDEX "subject_namespace_key_deleted_at" ON "subjects" ("namespace", "key", "deleted_at") WHERE (deleted_at IS NULL);
