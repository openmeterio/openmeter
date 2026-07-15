-- reverse: modify "ledger_credit_void_records" table
ALTER TABLE "ledger_credit_void_records" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: modify "ledger_breakage_records" table
ALTER TABLE "ledger_breakage_records" ALTER COLUMN "currency" TYPE character varying(3);
-- reverse: create index "ledger_tx_groups_idempotency_scope" to table: "ledger_transaction_groups"
DROP INDEX "ledger_tx_groups_idempotency_scope";
-- reverse: modify "ledger_transaction_groups" table
ALTER TABLE "ledger_transaction_groups" DROP CONSTRAINT "ledger_tx_group_idempotency_scope", DROP CONSTRAINT "ledger_tx_group_idempotency_fields", DROP COLUMN "input_fingerprint", DROP COLUMN "idempotency_key", DROP COLUMN "idempotency_scope";
-- reverse: modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" DROP COLUMN "custom_currency_version", DROP COLUMN "custom_currency_precision", DROP COLUMN "custom_currency_id", DROP COLUMN "exchange_source_currency";
