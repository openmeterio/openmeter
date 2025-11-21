-- modify "meters" table
ALTER TABLE "meters" ADD COLUMN "annotations" jsonb NULL;
-- create index "meter_annotations" to table: "meters"
CREATE INDEX "meter_annotations" ON "meters" USING gin ("annotations");
