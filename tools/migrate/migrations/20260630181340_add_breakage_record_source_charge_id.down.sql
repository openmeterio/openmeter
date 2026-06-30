-- reverse: create index "ledgerbreakagerecord_namespace_source_charge_id" to table: "ledger_breakage_records"
DROP INDEX "ledgerbreakagerecord_namespace_source_charge_id";
-- reverse: modify "ledger_breakage_records" table
ALTER TABLE "ledger_breakage_records" DROP COLUMN "source_charge_id";
