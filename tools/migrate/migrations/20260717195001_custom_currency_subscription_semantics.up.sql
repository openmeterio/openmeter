-- modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_item_currency_has_price", ADD CONSTRAINT "subscription_item_currency_has_price" CHECK (((price IS NULL) AND (currency IS NULL) AND (custom_currency_id IS NULL)) OR ((price IS NOT NULL) AND ((currency IS NOT NULL) OR (custom_currency_id IS NOT NULL))));
-- modify "subscriptions" table
ALTER TABLE "subscriptions" ADD COLUMN "cost_basis_mode" character varying NOT NULL DEFAULT 'dynamic';
-- create "subscription_cost_basis_pins" table
CREATE TABLE "subscription_cost_basis_pins" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "invoice_currency" character varying NOT NULL,
  "cost_basis_id" character(26) NOT NULL,
  "custom_currency_id" character(26) NOT NULL,
  "subscription_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_cost_basis_pins_currency_cost_bases_subscription_p" FOREIGN KEY ("cost_basis_id") REFERENCES "currency_cost_bases" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "subscription_cost_basis_pins_custom_currencies_subscription_cos" FOREIGN KEY ("custom_currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "subscription_cost_basis_pins_subscriptions_cost_basis_pins" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "subscriptioncostbasispin_cost_basis_id" to table: "subscription_cost_basis_pins"
CREATE INDEX "subscriptioncostbasispin_cost_basis_id" ON "subscription_cost_basis_pins" ("cost_basis_id");
-- create index "subscriptioncostbasispin_custom_currency_id" to table: "subscription_cost_basis_pins"
CREATE INDEX "subscriptioncostbasispin_custom_currency_id" ON "subscription_cost_basis_pins" ("custom_currency_id");
-- create index "subscriptioncostbasispin_id" to table: "subscription_cost_basis_pins"
CREATE UNIQUE INDEX "subscriptioncostbasispin_id" ON "subscription_cost_basis_pins" ("id");
-- create index "subscriptioncostbasispin_namespace" to table: "subscription_cost_basis_pins"
CREATE INDEX "subscriptioncostbasispin_namespace" ON "subscription_cost_basis_pins" ("namespace");
-- create index "subscriptioncostbasispin_namespace_subscription_id_custom_curre" to table: "subscription_cost_basis_pins"
CREATE UNIQUE INDEX "subscriptioncostbasispin_namespace_subscription_id_custom_curre" ON "subscription_cost_basis_pins" ("namespace", "subscription_id", "custom_currency_id", "invoice_currency");
-- create index "subscriptioncostbasispin_subscription_id" to table: "subscription_cost_basis_pins"
CREATE INDEX "subscriptioncostbasispin_subscription_id" ON "subscription_cost_basis_pins" ("subscription_id");
