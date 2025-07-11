-- reverse: create index "customer_annotations" to table: "customers"
DROP INDEX "customer_annotations";
-- reverse: modify "customers" table
ALTER TABLE "customers" DROP COLUMN "annotations";
