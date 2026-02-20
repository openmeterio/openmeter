-- create "tax_codes" table
CREATE TABLE "tax_codes" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "key" character varying NOT NULL,
  "app_mappings" jsonb NULL,
  PRIMARY KEY ("id")
);
-- create index "taxcode_id" to table: "tax_codes"
CREATE UNIQUE INDEX "taxcode_id" ON "tax_codes" ("id");
-- create index "taxcode_namespace" to table: "tax_codes"
CREATE INDEX "taxcode_namespace" ON "tax_codes" ("namespace");
-- create index "taxcode_namespace_id" to table: "tax_codes"
CREATE UNIQUE INDEX "taxcode_namespace_id" ON "tax_codes" ("namespace", "id");
-- create index "taxcode_namespace_key_deleted_at" to table: "tax_codes"
CREATE UNIQUE INDEX "taxcode_namespace_key_deleted_at" ON "tax_codes" ("namespace", "key", "deleted_at");
-- create index "taxcode_namespace_key" to table: "tax_codes"
CREATE UNIQUE INDEX "taxcode_namespace_key"
  ON "tax_codes" ("namespace", "key")
  WHERE "deleted_at" IS NULL;