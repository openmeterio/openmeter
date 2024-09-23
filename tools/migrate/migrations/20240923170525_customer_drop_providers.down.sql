-- reverse: modify "customers" table
ALTER TABLE "customers" ADD COLUMN "payment_provider" character varying NULL, ADD COLUMN "invoicing_provider" character varying NULL, ADD COLUMN "tax_provider" character varying NULL;
