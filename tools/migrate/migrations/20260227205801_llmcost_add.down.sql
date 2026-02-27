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
-- reverse: modify "features" table
ALTER TABLE "features" DROP COLUMN "unit_cost_llm_token_type", DROP COLUMN "unit_cost_llm_token_type_property", DROP COLUMN "unit_cost_llm_model", DROP COLUMN "unit_cost_llm_model_property", DROP COLUMN "unit_cost_llm_provider", DROP COLUMN "unit_cost_llm_provider_property", DROP COLUMN "unit_cost_manual_amount", DROP COLUMN "unit_cost_type";
