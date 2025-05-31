-- drop index "subject_key_organization_subjects" to table: "subjects"
DROP INDEX "subject_key_organization_subjects";

-- modify "subjects" table
-- note that alter table doesn't accept multiple column changes with renaming
-- atlas:nolint BC102
ALTER TABLE
    "subjects" RENAME COLUMN "organization_subjects" TO "namespace";

-- atlas:nolint DS103 BC102
ALTER TABLE
    "subjects" DROP COLUMN "current_period_start",
    DROP COLUMN "current_period_end";

-- create index "subject_namespace" to table: "subjects"
CREATE INDEX "subject_namespace" ON "subjects" ("namespace");

-- create index "subject_key_namespace" to table: "subjects"
CREATE UNIQUE INDEX "subject_key_namespace" ON "subjects" ("key", "namespace");