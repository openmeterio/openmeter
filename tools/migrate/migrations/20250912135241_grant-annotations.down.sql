-- reverse: create index "grant_annotations" to table: "grants"
DROP INDEX "grant_annotations";
-- reverse: modify "grants" table
ALTER TABLE "grants" DROP COLUMN "annotations";
