-- modify "billing_invoices" table
CREATE SEQUENCE "tmp_billing_sequence_numbers_namespace_scope" START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
-- let's add sane default values to the existing rows (INV-1, INV-2, ...)
UPDATE "billing_invoices" SET "number" = 'INV-' || nextval('tmp_billing_sequence_numbers_namespace_scope') WHERE "number" IS NULL;
DROP SEQUENCE "tmp_billing_sequence_numbers_namespace_scope";

ALTER TABLE "billing_invoices" ALTER COLUMN "number" SET NOT NULL;
-- create "billing_sequence_numbers" table
CREATE TABLE "billing_sequence_numbers" (
  "id" bigint NOT NULL GENERATED BY DEFAULT AS IDENTITY,
  "namespace" character varying NOT NULL,
  "scope" character varying NOT NULL,
  "last" numeric NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "billingsequencenumbers_namespace" to table: "billing_sequence_numbers"
CREATE INDEX "billingsequencenumbers_namespace" ON "billing_sequence_numbers" ("namespace");
-- create index "billingsequencenumbers_namespace_scope" to table: "billing_sequence_numbers"
CREATE UNIQUE INDEX "billingsequencenumbers_namespace_scope" ON "billing_sequence_numbers" ("namespace", "scope");