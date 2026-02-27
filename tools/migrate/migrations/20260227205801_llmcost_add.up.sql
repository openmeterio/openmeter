-- modify "features" table
ALTER TABLE "features" ADD COLUMN "unit_cost_type" character varying NULL, ADD COLUMN "unit_cost_manual_amount" numeric NULL, ADD COLUMN "unit_cost_llm_provider_property" character varying NULL, ADD COLUMN "unit_cost_llm_provider" character varying NULL, ADD COLUMN "unit_cost_llm_model_property" character varying NULL, ADD COLUMN "unit_cost_llm_model" character varying NULL, ADD COLUMN "unit_cost_llm_token_type_property" character varying NULL, ADD COLUMN "unit_cost_llm_token_type" character varying NULL;
-- create "llm_cost_prices" table
CREATE TABLE "llm_cost_prices" (
  "id" character(26) NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "namespace" character varying NULL,
  "provider" character varying NOT NULL,
  "model_id" character varying NOT NULL,
  "model_name" character varying NOT NULL DEFAULT '',
  "input_per_token" numeric NOT NULL,
  "output_per_token" numeric NOT NULL,
  "input_cached_per_token" numeric NOT NULL,
  "reasoning_per_token" numeric NOT NULL,
  "cache_write_per_token" numeric NOT NULL,
  "currency" character varying NOT NULL DEFAULT 'USD',
  "source" character varying NOT NULL,
  "source_prices" jsonb NULL,
  "effective_from" timestamptz NOT NULL,
  "effective_to" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- create index "llmcostprice_id" to table: "llm_cost_prices"
CREATE UNIQUE INDEX "llmcostprice_id" ON "llm_cost_prices" ("id");
-- create index "llmcostprice_namespace_provider_model_id" to table: "llm_cost_prices"
CREATE INDEX "llmcostprice_namespace_provider_model_id" ON "llm_cost_prices" ("namespace", "provider", "model_id") WHERE (deleted_at IS NULL);
-- create index "llmcostprice_provider_model_id" to table: "llm_cost_prices"
CREATE INDEX "llmcostprice_provider_model_id" ON "llm_cost_prices" ("provider", "model_id") WHERE ((deleted_at IS NULL) AND (namespace IS NULL));
-- create index "llmcostprice_provider_model_id_namespace_effective_from" to table: "llm_cost_prices"
CREATE UNIQUE INDEX "llmcostprice_provider_model_id_namespace_effective_from" ON "llm_cost_prices" ("provider", "model_id", "namespace", "effective_from") WHERE (deleted_at IS NULL);
