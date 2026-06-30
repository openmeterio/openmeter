-- reverse: create index "ledgerentry_namespace_spend_charge_id" to table: "ledger_entries"
DROP INDEX "ledgerentry_namespace_spend_charge_id";
-- reverse: create index "ledgerentry_namespace_source_charge_id_spend_charge_id" to table: "ledger_entries"
DROP INDEX "ledgerentry_namespace_source_charge_id_spend_charge_id";
-- reverse: create index "ledgerentry_namespace_source_charge_id" to table: "ledger_entries"
DROP INDEX "ledgerentry_namespace_source_charge_id";
-- reverse: modify "ledger_entries" table
ALTER TABLE "ledger_entries" DROP COLUMN "spend_charge_id", DROP COLUMN "source_charge_id";
