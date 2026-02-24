-- reverse: create index "currencycostbasis_namespace_custom_currency_id_fiat_code_effect" to table: "currency_cost_bases"
DROP INDEX "currencycostbasis_namespace_custom_currency_id_fiat_code_effect";
-- reverse: create index "currencycostbasis_namespace" to table: "currency_cost_bases"
DROP INDEX "currencycostbasis_namespace";
-- reverse: create index "currencycostbasis_id" to table: "currency_cost_bases"
DROP INDEX "currencycostbasis_id";
-- reverse: create "currency_cost_bases" table
DROP TABLE "currency_cost_bases";
-- reverse: create index "customcurrency_namespace_code" to table: "custom_currencies"
DROP INDEX "customcurrency_namespace_code";
-- reverse: create index "customcurrency_namespace" to table: "custom_currencies"
DROP INDEX "customcurrency_namespace";
-- reverse: create index "customcurrency_id" to table: "custom_currencies"
DROP INDEX "customcurrency_id";
-- reverse: create "custom_currencies" table
DROP TABLE "custom_currencies";
