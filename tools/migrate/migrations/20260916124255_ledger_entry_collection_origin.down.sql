-- reverse: create index "ledgerentry_namespace_collection_origin_id" to table: "ledger_entries"
DROP INDEX "ledgerentry_namespace_collection_origin_id";
-- reverse: modify "ledger_entries" table
ALTER TABLE "ledger_entries" DROP COLUMN "collection_origin_id";
