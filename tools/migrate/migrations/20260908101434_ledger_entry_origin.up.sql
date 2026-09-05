-- modify "ledger_entries" table
ALTER TABLE "ledger_entries" ADD COLUMN "origin_id" character(26) NULL;
-- create index "ledgerentry_namespace_origin_id" to table: "ledger_entries"
CREATE INDEX "ledgerentry_namespace_origin_id" ON "ledger_entries" ("namespace", "origin_id");
