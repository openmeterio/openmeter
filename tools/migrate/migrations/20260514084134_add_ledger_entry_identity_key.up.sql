-- modify "ledger_entries" table
ALTER TABLE "ledger_entries" ADD COLUMN "identity_key" character varying NOT NULL DEFAULT '';
-- create index "ledgerentry_transaction_id_sub_account_id_identity_key" to table: "ledger_entries"
CREATE UNIQUE INDEX "ledgerentry_transaction_id_sub_account_id_identity_key" ON "ledger_entries" ("transaction_id", "sub_account_id", "identity_key");
