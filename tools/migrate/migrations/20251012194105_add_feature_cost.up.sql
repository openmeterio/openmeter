-- modify "features" table
ALTER TABLE "features" ADD COLUMN "cost_kind" character varying NULL, ADD COLUMN "cost_currency" character varying(3) NULL, ADD COLUMN "cost_unit_amount" numeric NULL, ADD COLUMN "cost_provider_id" character varying NULL;
