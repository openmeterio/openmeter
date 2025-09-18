-- modify "grants" table
ALTER TABLE "grants" ADD COLUMN "annotations" jsonb NULL;

-- NOTE: metadata will be dropped later
-- NOTE: technically, this might parse the values from strings to other primitive types when we later parse this.
-- convert metadata to annotations: all values will be converted to strings
UPDATE "grants" SET "annotations" = "metadata";


-- create index "grant_annotations" to table: "grants"
CREATE INDEX "grant_annotations" ON "grants" USING gin ("annotations");
