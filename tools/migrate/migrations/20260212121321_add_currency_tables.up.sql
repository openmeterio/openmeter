-- create "custom_currencies" table
CREATE TABLE "custom_currencies" (
  "id" character(26) NOT NULL,
  "created_at" timestamptz NOT NULL,
  "code" character varying NOT NULL,
  "name" character varying NOT NULL,
  "symbol" character varying NOT NULL,
  "smallest_denomination" smallint NOT NULL DEFAULT 2,
  PRIMARY KEY ("id")
);
-- create index "custom_currencies_code_key" to table: "custom_currencies"
CREATE UNIQUE INDEX "custom_currencies_code_key" ON "custom_currencies" ("code");
-- create index "customcurrency_id" to table: "custom_currencies"
CREATE UNIQUE INDEX "customcurrency_id" ON "custom_currencies" ("id");
-- create "currency_cost_bases" table
CREATE TABLE "currency_cost_bases" (
  "id" character(26) NOT NULL,
  "created_at" timestamptz NOT NULL,
  "fiat_code" character varying NOT NULL,
  "rate" numeric NOT NULL,
  "effective_from" timestamptz NOT NULL,
  "custom_currency_cost_basis_history" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "currency_cost_bases_custom_currencies_cost_basis_history" FOREIGN KEY ("custom_currency_cost_basis_history") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "currencycostbasis_fiat_code_effective_from_custom_currency_cost" to table: "currency_cost_bases"
CREATE UNIQUE INDEX "currencycostbasis_fiat_code_effective_from_custom_currency_cost" ON "currency_cost_bases" ("fiat_code", "effective_from", "custom_currency_cost_basis_history");
-- create index "currencycostbasis_id" to table: "currency_cost_bases"
CREATE UNIQUE INDEX "currencycostbasis_id" ON "currency_cost_bases" ("id");
