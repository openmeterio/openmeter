-- Normalize provider names in llm_cost_prices to match updated NormalizeProvider() logic.
-- Hosting providers (azure, bedrock, vertex_ai) are kept separate from model vendors
-- (openai, anthropic, google) because pricing can differ.
--
-- The unique index is on (provider, model_id, namespace, effective_from) WHERE deleted_at IS NULL,
-- so deduplication must match on all four columns.

-- Step 1: For each alias group, soft-delete rows that would collide with an existing
-- canonical row on the full unique key (canonical_provider, model_id, namespace, effective_from).
-- This handles alias→canonical conflicts.
UPDATE llm_cost_prices AS dup
SET deleted_at = NOW()
WHERE deleted_at IS NULL
  AND provider IN ('nano-gpt', 'nano_gpt')
  AND EXISTS (
    SELECT 1 FROM llm_cost_prices AS canon
    WHERE canon.provider = 'nanogpt'
      AND canon.model_id = dup.model_id
      AND canon.namespace IS NOT DISTINCT FROM dup.namespace
      AND canon.effective_from = dup.effective_from
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
      AND canon.effective_from = dup.effective_from
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
      AND canon.effective_from = dup.effective_from
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
      AND canon.effective_from = dup.effective_from
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
      AND canon.effective_from = dup.effective_from
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
      AND canon.effective_from = dup.effective_from
      AND canon.deleted_at IS NULL
  );

-- Step 2: Handle alias-vs-alias collisions within the same group.
-- E.g., if both nano-gpt and nano_gpt exist for the same (model_id, namespace, effective_from),
-- keep only one (the row with the smallest id) and soft-delete the rest.
WITH dupes AS (
  SELECT id,
         ROW_NUMBER() OVER (
           PARTITION BY
             CASE
               WHEN provider IN ('nano-gpt', 'nano_gpt') THEN 'nanogpt'
               WHEN provider LIKE 'vertex\_ai-%' OR provider LIKE 'vertex\_ai\_%' THEN 'vertex_ai'
               WHEN provider = 'x-ai' THEN 'xai'
               WHEN provider = 'azure_ai' THEN 'azure'
               WHEN provider = 'bedrock_converse' THEN 'bedrock'
               WHEN provider = 'gemini' THEN 'google'
               ELSE provider
             END,
             model_id, namespace, effective_from
           ORDER BY id
         ) AS rn
  FROM llm_cost_prices
  WHERE deleted_at IS NULL
    AND provider IN ('nano-gpt', 'nano_gpt', 'x-ai', 'azure_ai', 'bedrock_converse', 'gemini')
    OR (deleted_at IS NULL AND (provider LIKE 'vertex\_ai-%' OR provider LIKE 'vertex\_ai\_%') AND provider != 'vertex_ai')
)
UPDATE llm_cost_prices
SET deleted_at = NOW()
WHERE id IN (SELECT id FROM dupes WHERE rn > 1);

-- Step 3: Rename remaining alias providers to their canonical names.
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

-- Step 4: Strip provider prefixes from model_name (e.g., "azure/gpt-3.5-turbo" → "gpt-3.5-turbo").
UPDATE llm_cost_prices
SET model_name = substring(model_name FROM position('/' IN model_name) + 1),
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND model_name LIKE '%/%'
  AND position('/' IN model_name) > 1;
