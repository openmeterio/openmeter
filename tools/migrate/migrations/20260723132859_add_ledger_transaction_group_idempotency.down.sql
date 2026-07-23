-- reverse: create index "ledger_tx_groups_namespace_idempotency_key" to table: "ledger_transaction_groups"
DROP INDEX "ledger_tx_groups_namespace_idempotency_key";
-- reverse: modify "ledger_transaction_groups" table
ALTER TABLE "ledger_transaction_groups" DROP CONSTRAINT "ledger_tx_group_idempotency_pair", DROP COLUMN "input_fingerprint", DROP COLUMN "idempotency_key";
