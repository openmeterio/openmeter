-- reverse: drop "ledger_dimensions" table
CREATE TABLE "ledger_dimensions" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "dimension_key" character varying NOT NULL,
  "dimension_value" character varying NOT NULL,
  "dimension_display_value" character varying NOT NULL,
  PRIMARY KEY ("id")
);
CREATE INDEX "ledgerdimension_annotations" ON "ledger_dimensions" USING gin ("annotations");
CREATE UNIQUE INDEX "ledgerdimension_id" ON "ledger_dimensions" ("id");
CREATE INDEX "ledgerdimension_namespace" ON "ledger_dimensions" ("namespace");
CREATE UNIQUE INDEX "ledgerdimension_namespace_dimension_key_dimension_value" ON "ledger_dimensions" ("namespace", "dimension_key", "dimension_value");
CREATE UNIQUE INDEX "ledgerdimension_namespace_id" ON "ledger_dimensions" ("namespace", "id");
-- reverse: modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" DROP COLUMN "credit_priority", DROP COLUMN "features", DROP COLUMN "tax_code", DROP COLUMN "currency", ADD COLUMN "credit_priority_dimension_id" character(26) NULL, ADD COLUMN "features_dimension_id" character(26) NULL, ADD COLUMN "tax_code_dimension_id" character(26) NULL, ADD COLUMN "currency_dimension_id" character(26) NOT NULL, ADD COLUMN "ledger_dimension_sub_account_routes" character(26) NULL;
