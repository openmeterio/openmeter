-- drop index "subject_key_organization_subjects" to table: "subjects"
DROP INDEX "subject_key_organization_subjects";

-- modify "subjects" table
-- atlas:nolint BC102
ALTER TABLE "subjects" RENAME COLUMN "organization_subjects" TO "namespace";

-- atlas:nolint DS103 BC102
ALTER TABLE "subjects"
    DROP COLUMN "current_period_start",
    DROP COLUMN "current_period_end";

-- create index "subject_namespace" to table: "subjects"
CREATE INDEX IF NOT EXISTS "subject_namespace" ON "subjects" ("namespace");

-- create index "subject_key_namespace" to table: "subjects"
-- atlas:nolint MF101
CREATE UNIQUE INDEX IF NOT EXISTS"subject_key_namespace" ON "subjects" ("key", "namespace");
