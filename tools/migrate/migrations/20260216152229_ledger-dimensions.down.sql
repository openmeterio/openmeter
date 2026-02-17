-- reverse: create index "ledgerdimension_namespace_dimension_key_dimension_value" to table: "ledger_dimensions"
DROP INDEX "ledgerdimension_namespace_dimension_key_dimension_value";
-- reverse: modify "ledger_dimensions" table
ALTER TABLE "ledger_dimensions" DROP COLUMN "dimension_display_value";
-- reverse: drop index "ledgerdimension_namespace_dimension_key_dimension_value" from table: "ledger_dimensions"
CREATE INDEX "ledgerdimension_namespace_dimension_key_dimension_value" ON "ledger_dimensions" ("namespace", "dimension_key", "dimension_value");
