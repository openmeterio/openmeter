-- create "custom_currencies" table
CREATE TABLE "custom_currencies" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "code" character varying NOT NULL,
  "name" character varying NOT NULL,
  "symbol" character varying NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "customcurrency_id" to table: "custom_currencies"
CREATE UNIQUE INDEX "customcurrency_id" ON "custom_currencies" ("id");
-- create index "customcurrency_namespace" to table: "custom_currencies"
CREATE INDEX "customcurrency_namespace" ON "custom_currencies" ("namespace");
-- create index "customcurrency_namespace_code" to table: "custom_currencies"
CREATE UNIQUE INDEX "customcurrency_namespace_code" ON "custom_currencies" ("namespace", "code") WHERE (deleted_at IS NULL);
-- create index "customcurrency_namespace_id" to table: "custom_currencies"
CREATE UNIQUE INDEX "customcurrency_namespace_id" ON "custom_currencies" ("namespace", "id");
-- create "currency_cost_bases" table
CREATE TABLE "currency_cost_bases" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "fiat_code" character varying NOT NULL,
  "rate" numeric NOT NULL,
  "effective_from" timestamptz NOT NULL,
  "custom_currency_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "currency_cost_bases_custom_currencies_cost_basis_history" FOREIGN KEY ("custom_currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "currencycostbasis_id" to table: "currency_cost_bases"
CREATE UNIQUE INDEX "currencycostbasis_id" ON "currency_cost_bases" ("id");
-- create index "currencycostbasis_namespace" to table: "currency_cost_bases"
CREATE INDEX "currencycostbasis_namespace" ON "currency_cost_bases" ("namespace");
-- create index "currencycostbasis_namespace_custom_currency_id_fiat_code_effect" to table: "currency_cost_bases"
CREATE UNIQUE INDEX "currencycostbasis_namespace_custom_currency_id_fiat_code_effect" ON "currency_cost_bases" ("namespace", "custom_currency_id", "fiat_code", "effective_from") WHERE (deleted_at IS NULL);
