-- reverse: create index "ledger_tx_groups_idempotency_scope" to table: "ledger_transaction_groups"
DROP INDEX "ledger_tx_groups_idempotency_scope";
-- reverse: modify "ledger_transaction_groups" table
ALTER TABLE "ledger_transaction_groups" DROP CONSTRAINT "ledger_tx_group_idempotency_scope", DROP CONSTRAINT "ledger_tx_group_idempotency_fields", DROP COLUMN "input_fingerprint", DROP COLUMN "idempotency_key", DROP COLUMN "idempotency_scope";
-- reverse: modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" DROP COLUMN "exchange_source_currency";
