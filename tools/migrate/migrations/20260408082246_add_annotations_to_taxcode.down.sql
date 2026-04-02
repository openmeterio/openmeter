-- reverse: create index "taxcode_annotations" to table: "tax_codes"
DROP INDEX "taxcode_annotations";
-- reverse: modify "tax_codes" table
ALTER TABLE "tax_codes" DROP COLUMN "annotations";
