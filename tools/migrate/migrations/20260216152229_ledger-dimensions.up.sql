-- drop index "ledgerdimension_namespace_dimension_key_dimension_value" from table: "ledger_dimensions"
DROP INDEX "ledgerdimension_namespace_dimension_key_dimension_value";
-- modify "ledger_dimensions" table
ALTER TABLE "ledger_dimensions" ADD COLUMN "dimension_display_value" character varying NOT NULL;
-- create index "ledgerdimension_namespace_dimension_key_dimension_value" to table: "ledger_dimensions"
CREATE UNIQUE INDEX "ledgerdimension_namespace_dimension_key_dimension_value" ON "ledger_dimensions" ("namespace", "dimension_key", "dimension_value");
