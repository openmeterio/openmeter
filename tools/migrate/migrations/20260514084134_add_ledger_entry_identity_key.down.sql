-- reverse: create index "ledgerentry_transaction_id_sub_account_id_identity_key" to table: "ledger_entries"
DROP INDEX "ledgerentry_transaction_id_sub_account_id_identity_key";
-- reverse: modify "ledger_entries" table
ALTER TABLE "ledger_entries" DROP COLUMN "identity_key";
