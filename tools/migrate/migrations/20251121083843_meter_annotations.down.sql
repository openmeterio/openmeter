-- reverse: create index "meter_annotations" to table: "meters"
DROP INDEX "meter_annotations";
-- reverse: modify "meters" table
ALTER TABLE "meters" DROP COLUMN "annotations";
