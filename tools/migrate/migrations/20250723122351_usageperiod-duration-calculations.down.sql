-- reverse: create index "usagereset_annotations" to table: "usage_resets"
DROP INDEX "usagereset_annotations";
-- reverse: modify "usage_resets" table
ALTER TABLE "usage_resets" DROP COLUMN "annotations";
