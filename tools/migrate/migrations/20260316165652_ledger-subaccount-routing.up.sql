-- create "ledger_sub_account_routes" table
CREATE TABLE "ledger_sub_account_routes" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "routing_key_version" character varying NOT NULL,
  "routing_key" character varying NOT NULL,
  "account_id" character(26) NOT NULL,
  "ledger_dimension_sub_account_routes" character(26) NULL,
  "currency_dimension_id" character(26) NOT NULL,
  "tax_code_dimension_id" character(26) NULL,
  "features_dimension_id" character(26) NULL,
  "credit_priority_dimension_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "ledger_sub_account_routes_ledger_accounts_sub_account_routes" FOREIGN KEY ("account_id") REFERENCES "ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "ledger_sub_account_routes_ledger_dimensions_credit_priority_sub" FOREIGN KEY ("credit_priority_dimension_id") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "ledger_sub_account_routes_ledger_dimensions_currency_sub_accoun" FOREIGN KEY ("currency_dimension_id") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "ledger_sub_account_routes_ledger_dimensions_features_sub_accoun" FOREIGN KEY ("features_dimension_id") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "ledger_sub_account_routes_ledger_dimensions_sub_account_routes" FOREIGN KEY ("ledger_dimension_sub_account_routes") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "ledger_sub_account_routes_ledger_dimensions_tax_code_sub_accoun" FOREIGN KEY ("tax_code_dimension_id") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "ledgersubaccountroute_id" to table: "ledger_sub_account_routes"
CREATE UNIQUE INDEX "ledgersubaccountroute_id" ON "ledger_sub_account_routes" ("id");
-- create index "ledgersubaccountroute_namespace" to table: "ledger_sub_account_routes"
CREATE INDEX "ledgersubaccountroute_namespace" ON "ledger_sub_account_routes" ("namespace");
-- create index "ledgersubaccountroute_namespace_account_id_routing_key_version_" to table: "ledger_sub_account_routes"
CREATE UNIQUE INDEX "ledgersubaccountroute_namespace_account_id_routing_key_version_" ON "ledger_sub_account_routes" ("namespace", "account_id", "routing_key_version", "routing_key");
-- modify "ledger_sub_accounts" table
-- atlas:nolint DS103 MF103
ALTER TABLE "ledger_sub_accounts" DROP COLUMN "ledger_dimension_sub_accounts", DROP COLUMN "currency_dimension_id", DROP COLUMN "tax_code_dimension_id", DROP COLUMN "features_dimension_id", DROP COLUMN "credit_priority_dimension_id", ADD COLUMN "route_id" character(26) NOT NULL, ADD CONSTRAINT "ledger_sub_accounts_ledger_sub_account_routes_sub_accounts" FOREIGN KEY ("route_id") REFERENCES "ledger_sub_account_routes" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- create index "ledgersubaccount_namespace_account_id_route_id" to table: "ledger_sub_accounts"
CREATE UNIQUE INDEX "ledgersubaccount_namespace_account_id_route_id" ON "ledger_sub_accounts" ("namespace", "account_id", "route_id");
