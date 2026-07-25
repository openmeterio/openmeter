-- modify "ledger_breakage_records" table
ALTER TABLE "ledger_breakage_records" ADD COLUMN "source_charge_id" character(26) NULL;
-- create index "ledgerbreakagerecord_namespace_source_charge_id" to table: "ledger_breakage_records"
CREATE INDEX "ledgerbreakagerecord_namespace_source_charge_id" ON "ledger_breakage_records" ("namespace", "source_charge_id");
