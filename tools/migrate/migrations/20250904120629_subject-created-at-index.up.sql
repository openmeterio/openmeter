-- create index "subject_created_at_id" to table: "subjects"
CREATE INDEX "subject_created_at_id" ON "subjects" ("created_at", "id");
