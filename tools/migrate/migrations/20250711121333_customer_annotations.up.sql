-- modify "customers" table
ALTER TABLE "customers" ADD COLUMN "annotations" jsonb NULL;
-- create index "customer_annotations" to table: "customers"
CREATE INDEX "customer_annotations" ON "customers" USING gin ("annotations");
