-- modify "ledger_entries" table
ALTER TABLE "ledger_entries" ADD COLUMN "source_charge_id" character(26) NULL, ADD COLUMN "spend_charge_id" character(26) NULL;
-- create index "ledgerentry_namespace_source_charge_id" to table: "ledger_entries"
CREATE INDEX "ledgerentry_namespace_source_charge_id" ON "ledger_entries" ("namespace", "source_charge_id");
-- create index "ledgerentry_namespace_source_charge_id_spend_charge_id" to table: "ledger_entries"
CREATE INDEX "ledgerentry_namespace_source_charge_id_spend_charge_id" ON "ledger_entries" ("namespace", "source_charge_id", "spend_charge_id");
-- create index "ledgerentry_namespace_spend_charge_id" to table: "ledger_entries"
CREATE INDEX "ledgerentry_namespace_spend_charge_id" ON "ledger_entries" ("namespace", "spend_charge_id");
