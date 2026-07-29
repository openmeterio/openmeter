-- modify "ledger_breakage_records" table
ALTER TABLE "ledger_breakage_records" ALTER COLUMN "currency" TYPE character varying;
-- modify "ledger_credit_void_records" table
ALTER TABLE "ledger_credit_void_records" ALTER COLUMN "currency" TYPE character varying;
-- modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" ADD COLUMN "cost_basis_currency" character varying NULL, ADD COLUMN "custom_currency_id" character(26) NULL, ADD COLUMN "custom_currency_precision" bigint NULL, ADD COLUMN "custom_currency_version" bigint NULL;
