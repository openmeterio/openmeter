-- Normalize provider names in llm_cost_prices to match updated NormalizeProvider() logic.
-- Hosting providers (azure, bedrock, vertex_ai) are kept separate from model vendors
-- (openai, anthropic, google) because pricing can differ.

-- Step 1: Soft-delete rows that would become duplicates after provider renaming.
-- For each (new_provider, model_id, namespace) that already has a row
-- under the canonical provider, soft-delete the row with the alias provider.
UPDATE llm_cost_prices AS dup
SET deleted_at = NOW()
WHERE deleted_at IS NULL
  AND provider IN ('nano-gpt', 'nano_gpt')
  AND EXISTS (
    SELECT 1 FROM llm_cost_prices AS canon
    WHERE canon.provider = 'nanogpt'
      AND canon.model_id = dup.model_id
      AND canon.namespace IS NOT DISTINCT FROM dup.namespace
      AND canon.deleted_at IS NULL
  );

UPDATE llm_cost_prices AS dup
SET deleted_at = NOW()
WHERE deleted_at IS NULL
  AND (provider LIKE 'vertex\_ai-%' OR provider LIKE 'vertex\_ai\_%')
  AND provider != 'vertex_ai'
  AND EXISTS (
    SELECT 1 FROM llm_cost_prices AS canon
    WHERE canon.provider = 'vertex_ai'
      AND canon.model_id = dup.model_id
      AND canon.namespace IS NOT DISTINCT FROM dup.namespace
      AND canon.deleted_at IS NULL
  );

UPDATE llm_cost_prices AS dup
SET deleted_at = NOW()
WHERE deleted_at IS NULL
  AND provider = 'x-ai'
  AND EXISTS (
    SELECT 1 FROM llm_cost_prices AS canon
    WHERE canon.provider = 'xai'
      AND canon.model_id = dup.model_id
      AND canon.namespace IS NOT DISTINCT FROM dup.namespace
      AND canon.deleted_at IS NULL
  );

UPDATE llm_cost_prices AS dup
SET deleted_at = NOW()
WHERE deleted_at IS NULL
  AND provider = 'azure_ai'
  AND EXISTS (
    SELECT 1 FROM llm_cost_prices AS canon
    WHERE canon.provider = 'azure'
      AND canon.model_id = dup.model_id
      AND canon.namespace IS NOT DISTINCT FROM dup.namespace
      AND canon.deleted_at IS NULL
  );

UPDATE llm_cost_prices AS dup
SET deleted_at = NOW()
WHERE deleted_at IS NULL
  AND provider = 'bedrock_converse'
  AND EXISTS (
    SELECT 1 FROM llm_cost_prices AS canon
    WHERE canon.provider = 'bedrock'
      AND canon.model_id = dup.model_id
      AND canon.namespace IS NOT DISTINCT FROM dup.namespace
      AND canon.deleted_at IS NULL
  );

UPDATE llm_cost_prices AS dup
SET deleted_at = NOW()
WHERE deleted_at IS NULL
  AND provider = 'gemini'
  AND EXISTS (
    SELECT 1 FROM llm_cost_prices AS canon
    WHERE canon.provider = 'google'
      AND canon.model_id = dup.model_id
      AND canon.namespace IS NOT DISTINCT FROM dup.namespace
      AND canon.deleted_at IS NULL
  );

-- Step 2: Rename remaining alias providers to their canonical names.
UPDATE llm_cost_prices
SET provider = 'nanogpt', updated_at = NOW()
WHERE deleted_at IS NULL
  AND provider IN ('nano-gpt', 'nano_gpt');

UPDATE llm_cost_prices
SET provider = 'vertex_ai', updated_at = NOW()
WHERE deleted_at IS NULL
  AND (provider LIKE 'vertex\_ai-%' OR provider LIKE 'vertex\_ai\_%')
  AND provider != 'vertex_ai';

UPDATE llm_cost_prices
SET provider = 'xai', updated_at = NOW()
WHERE deleted_at IS NULL
  AND provider = 'x-ai';

UPDATE llm_cost_prices
SET provider = 'azure', updated_at = NOW()
WHERE deleted_at IS NULL
  AND provider = 'azure_ai';

UPDATE llm_cost_prices
SET provider = 'bedrock', updated_at = NOW()
WHERE deleted_at IS NULL
  AND provider = 'bedrock_converse';

UPDATE llm_cost_prices
SET provider = 'google', updated_at = NOW()
WHERE deleted_at IS NULL
  AND provider = 'gemini';

-- Step 3: Strip provider prefixes from model_name (e.g., "azure/gpt-3.5-turbo" → "gpt-3.5-turbo").
UPDATE llm_cost_prices
SET model_name = substring(model_name FROM position('/' IN model_name) + 1),
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND model_name LIKE '%/%'
  AND position('/' IN model_name) > 1;
