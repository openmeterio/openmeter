-- reverse: modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" DROP COLUMN "custom_currency_version", DROP COLUMN "custom_currency_precision", DROP COLUMN "custom_currency_id", DROP COLUMN "cost_basis_currency";
-- reverse: modify "ledger_credit_void_records" table
ALTER TABLE "ledger_credit_void_records" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: modify "ledger_breakage_records" table
ALTER TABLE "ledger_breakage_records" ALTER COLUMN "currency" TYPE character varying(3);
