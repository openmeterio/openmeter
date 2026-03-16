-- modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" DROP COLUMN "ledger_dimension_sub_account_routes", DROP COLUMN "currency_dimension_id", DROP COLUMN "tax_code_dimension_id", DROP COLUMN "features_dimension_id", DROP COLUMN "credit_priority_dimension_id", ADD COLUMN "currency" character varying NOT NULL, ADD COLUMN "tax_code" character varying NULL, ADD COLUMN "features" jsonb NULL, ADD COLUMN "credit_priority" bigint NULL;
-- drop "ledger_dimensions" table
DROP TABLE "ledger_dimensions";
