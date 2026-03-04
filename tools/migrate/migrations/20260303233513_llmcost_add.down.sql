-- reverse: create index "llmcostprice_provider_model_id_namespace_effective_from" to table: "llm_cost_prices"
DROP INDEX "llmcostprice_provider_model_id_namespace_effective_from";
-- reverse: create index "llmcostprice_provider_model_id" to table: "llm_cost_prices"
DROP INDEX "llmcostprice_provider_model_id";
-- reverse: create index "llmcostprice_namespace_provider_model_id" to table: "llm_cost_prices"
DROP INDEX "llmcostprice_namespace_provider_model_id";
-- reverse: create index "llmcostprice_id" to table: "llm_cost_prices"
DROP INDEX "llmcostprice_id";
-- reverse: create "llm_cost_prices" table
DROP TABLE "llm_cost_prices";
