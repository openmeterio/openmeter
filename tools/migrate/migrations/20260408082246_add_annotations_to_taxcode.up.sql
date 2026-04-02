-- modify "tax_codes" table
ALTER TABLE "tax_codes" ADD COLUMN "annotations" jsonb NULL;
-- create index "taxcode_annotations" to table: "tax_codes"
CREATE INDEX "taxcode_annotations" ON "tax_codes" USING gin ("annotations");
