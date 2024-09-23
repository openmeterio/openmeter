-- reverse: modify "customers" table
-- atlas:nolint DS103
ALTER TABLE "customers" ADD COLUMN "payment_provider" character varying NULL, ADD COLUMN "invoicing_provider" character varying NULL, ADD COLUMN "tax_provider" character varying NULL;
