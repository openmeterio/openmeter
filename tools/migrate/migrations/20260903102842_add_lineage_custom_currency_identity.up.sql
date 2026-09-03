-- modify "credit_realization_lineages" table
ALTER TABLE "credit_realization_lineages" ALTER COLUMN "currency" TYPE character varying(24), ADD COLUMN "custom_currency_id" character(26) NULL;
