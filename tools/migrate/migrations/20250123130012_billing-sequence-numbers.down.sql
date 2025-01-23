-- reverse: create index "billingsequencenumbers_namespace_scope" to table: "billing_sequence_numbers"
DROP INDEX "billingsequencenumbers_namespace_scope";
-- reverse: create index "billingsequencenumbers_namespace" to table: "billing_sequence_numbers"
DROP INDEX "billingsequencenumbers_namespace";
-- reverse: create "billing_sequence_numbers" table
DROP TABLE "billing_sequence_numbers";
-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" ALTER COLUMN "number" DROP NOT NULL;
