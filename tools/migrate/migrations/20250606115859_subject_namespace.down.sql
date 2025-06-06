-- modify "subjects" table
-- atlas:nolint BC102
ALTER TABLE "subjects" RENAME COLUMN "namespace" TO "organization_subjects";

-- create index "subject_key_organization_subjects" to table: "subjects"
CREATE UNIQUE INDEX "subject_key_organization_subjects" ON "subjects" ("key", "organization_subjects");

-- atlas:nolint DS103 BC102
ALTER TABLE "subjects"
    ADD COLUMN "current_period_start" timestamptz NULL,
    ADD COLUMN "current_period_end" timestamptz NULL;

-- drop index "subject_namespace" from table: "subjects"
DROP INDEX "subject_namespace";

-- drop index "subject_key_namespace" from table: "subjects"
DROP INDEX "subject_key_namespace";
