-- modify "ledger_transaction_groups" table
ALTER TABLE "ledger_transaction_groups" ADD CONSTRAINT "ledger_tx_group_idempotency_pair" CHECK ((idempotency_key IS NULL) = (input_fingerprint IS NULL)), ADD COLUMN "idempotency_key" character varying NULL, ADD COLUMN "input_fingerprint" character varying NULL;
-- create index "ledger_tx_groups_namespace_idempotency_key" to table: "ledger_transaction_groups"
CREATE UNIQUE INDEX "ledger_tx_groups_namespace_idempotency_key" ON "ledger_transaction_groups" ("idempotency_key", "namespace") WHERE (idempotency_key IS NOT NULL);
