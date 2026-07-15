-- modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" ADD COLUMN "exchange_source_currency" character varying NULL, ADD COLUMN "custom_currency_id" character(26) NULL, ADD COLUMN "custom_currency_precision" bigint NULL, ADD COLUMN "custom_currency_version" bigint NULL;
-- modify "ledger_breakage_records" table
ALTER TABLE "ledger_breakage_records" ALTER COLUMN "currency" TYPE character varying;
-- modify "ledger_credit_void_records" table
ALTER TABLE "ledger_credit_void_records" ALTER COLUMN "currency" TYPE character varying;
-- modify "ledger_transaction_groups" table
ALTER TABLE "ledger_transaction_groups" ADD CONSTRAINT "ledger_tx_group_idempotency_fields" CHECK (((idempotency_key IS NULL) = (input_fingerprint IS NULL)) AND ((idempotency_key IS NULL) = (idempotency_scope IS NULL))), ADD CONSTRAINT "ledger_tx_group_idempotency_scope" CHECK ((idempotency_scope IS NULL) OR ((idempotency_scope)::text = ((((octet_length((namespace)::text))::text || ':'::text) || (namespace)::text) || (idempotency_key)::text))), ADD COLUMN "idempotency_scope" character varying NULL, ADD COLUMN "idempotency_key" character varying NULL, ADD COLUMN "input_fingerprint" character varying NULL;
-- create index "ledger_tx_groups_idempotency_scope" to table: "ledger_transaction_groups"
CREATE UNIQUE INDEX "ledger_tx_groups_idempotency_scope" ON "ledger_transaction_groups" ("idempotency_scope") WHERE (idempotency_scope IS NOT NULL);
