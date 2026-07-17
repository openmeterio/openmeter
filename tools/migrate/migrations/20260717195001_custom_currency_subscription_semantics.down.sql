-- reverse: create index "subscriptioncostbasispin_subscription_id" to table: "subscription_cost_basis_pins"
DROP INDEX "subscriptioncostbasispin_subscription_id";
-- reverse: create index "subscriptioncostbasispin_namespace_subscription_id_custom_curre" to table: "subscription_cost_basis_pins"
DROP INDEX "subscriptioncostbasispin_namespace_subscription_id_custom_curre";
-- reverse: create index "subscriptioncostbasispin_namespace" to table: "subscription_cost_basis_pins"
DROP INDEX "subscriptioncostbasispin_namespace";
-- reverse: create index "subscriptioncostbasispin_id" to table: "subscription_cost_basis_pins"
DROP INDEX "subscriptioncostbasispin_id";
-- reverse: create index "subscriptioncostbasispin_custom_currency_id" to table: "subscription_cost_basis_pins"
DROP INDEX "subscriptioncostbasispin_custom_currency_id";
-- reverse: create index "subscriptioncostbasispin_cost_basis_id" to table: "subscription_cost_basis_pins"
DROP INDEX "subscriptioncostbasispin_cost_basis_id";
-- reverse: create "subscription_cost_basis_pins" table
DROP TABLE "subscription_cost_basis_pins";
-- reverse: modify "subscriptions" table
ALTER TABLE "subscriptions" DROP COLUMN "cost_basis_mode";
-- reverse: modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_item_currency_has_price", ADD CONSTRAINT "subscription_item_currency_has_price" CHECK ((price IS NOT NULL) OR ((currency IS NULL) AND (custom_currency_id IS NULL)));
