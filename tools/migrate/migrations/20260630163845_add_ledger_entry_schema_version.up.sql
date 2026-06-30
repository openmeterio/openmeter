-- modify "ledger_entries" table
ALTER TABLE "ledger_entries" ADD COLUMN "schema_version" bigint NOT NULL DEFAULT 1;
