-- reverse: create index "currencycostbasis_id" to table: "currency_cost_bases"
DROP INDEX "currencycostbasis_id";
-- reverse: create index "currencycostbasis_fiat_code_effective_from_custom_currency_cost" to table: "currency_cost_bases"
DROP INDEX "currencycostbasis_fiat_code_effective_from_custom_currency_cost";
-- reverse: create "currency_cost_bases" table
DROP TABLE "currency_cost_bases";
-- reverse: create index "customcurrency_id" to table: "custom_currencies"
DROP INDEX "customcurrency_id";
-- reverse: create index "custom_currencies_code_key" to table: "custom_currencies"
DROP INDEX "custom_currencies_code_key";
-- reverse: create "custom_currencies" table
DROP TABLE "custom_currencies";
