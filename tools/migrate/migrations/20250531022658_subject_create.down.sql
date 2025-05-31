-- reverse: create index "subject_key_organization_subjects" to table: "subjects"
DROP INDEX "subject_key_organization_subjects";

-- reverse: create index "subject_id" to table: "subjects"
DROP INDEX "subject_id";

-- reverse: create index "subject_display_name" to table: "subjects"
DROP INDEX "subject_display_name";

-- reverse: create "subjects" table
DROP TABLE "subjects";