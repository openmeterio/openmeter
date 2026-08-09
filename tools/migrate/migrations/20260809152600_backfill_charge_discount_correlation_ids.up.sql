-- Charge-owned discounts require stable correlation IDs for detailed-line lineage. Reuse the
-- newest charge-backed invoice-line ID when possible so pending and already materialized lines
-- keep their existing identity; generate a new ID only when no line can provide one.
BEGIN;

CREATE TEMPORARY TABLE charge_discount_correlation_ids ON COMMIT DROP AS
WITH candidates AS (
  SELECT
    l.charge_id,
    discount_type.name AS discount_type,
    l.ratecard_discounts #>> ARRAY[discount_type.name, 'correlationID'] AS correlation_id,
    0 AS source_priority,
    l.updated_at,
    l.id
  FROM billing_gathering_invoice_lines l
  CROSS JOIN (VALUES ('percentage'), ('usage')) AS discount_type(name)
  WHERE l.charge_id IS NOT NULL
    AND l.deleted_at IS NULL
    AND NULLIF(l.ratecard_discounts #>> ARRAY[discount_type.name, 'correlationID'], '') IS NOT NULL

  UNION ALL

  SELECT
    l.charge_id,
    discount_type.name AS discount_type,
    l.ratecard_discounts #>> ARRAY[discount_type.name, 'correlationID'] AS correlation_id,
    1 AS source_priority,
    l.updated_at,
    l.id
  FROM billing_invoice_lines l
  CROSS JOIN (VALUES ('percentage'), ('usage')) AS discount_type(name)
  WHERE l.charge_id IS NOT NULL
    AND l.deleted_at IS NULL
    AND NULLIF(l.ratecard_discounts #>> ARRAY[discount_type.name, 'correlationID'], '') IS NOT NULL
), ranked AS (
  SELECT
    candidates.*,
    row_number() OVER (
      PARTITION BY charge_id, discount_type
      ORDER BY source_priority, updated_at DESC, id DESC
    ) AS row_number
  FROM candidates
)
SELECT
  charge_id,
  max(correlation_id) FILTER (WHERE discount_type = 'percentage') AS percentage_correlation_id,
  max(correlation_id) FILTER (WHERE discount_type = 'usage') AS usage_correlation_id
FROM ranked
WHERE row_number = 1
GROUP BY charge_id;

CREATE OR REPLACE FUNCTION pg_temp.upsert_discount_correlation_id(
  discounts jsonb,
  discount_type text,
  correlation_id text
)
RETURNS jsonb
AS $$
  SELECT CASE
    WHEN jsonb_typeof(discounts -> discount_type) = 'object'
      AND COALESCE(discounts #>> ARRAY[discount_type, 'correlationID'], '') = ''
    THEN jsonb_set(
      discounts,
      ARRAY[discount_type, 'correlationID'],
      to_jsonb(correlation_id),
      true
    )
    ELSE discounts
  END
$$
LANGUAGE sql
IMMUTABLE;

UPDATE charge_usage_based charge
SET
  discounts = pg_temp.upsert_discount_correlation_id(
    pg_temp.upsert_discount_correlation_id(
      charge.discounts,
      'percentage',
      COALESCE(
        (SELECT ids.percentage_correlation_id FROM charge_discount_correlation_ids ids WHERE ids.charge_id = charge.id),
        om_func_generate_ulid()
      )
    ),
    'usage',
    COALESCE(
      (SELECT ids.usage_correlation_id FROM charge_discount_correlation_ids ids WHERE ids.charge_id = charge.id),
      om_func_generate_ulid()
    )
  ),
  updated_at = now()
WHERE (
    jsonb_typeof(charge.discounts -> 'percentage') = 'object'
    AND COALESCE(charge.discounts #>> '{percentage,correlationID}', '') = ''
  ) OR (
    jsonb_typeof(charge.discounts -> 'usage') = 'object'
    AND COALESCE(charge.discounts #>> '{usage,correlationID}', '') = ''
  );

UPDATE charge_usage_based_overrides override
SET discounts = pg_temp.upsert_discount_correlation_id(
  pg_temp.upsert_discount_correlation_id(
    override.discounts,
    'percentage',
    COALESCE(
      (SELECT ids.percentage_correlation_id FROM charge_discount_correlation_ids ids WHERE ids.charge_id = override.charge_id),
      om_func_generate_ulid()
    )
  ),
  'usage',
  COALESCE(
    (SELECT ids.usage_correlation_id FROM charge_discount_correlation_ids ids WHERE ids.charge_id = override.charge_id),
    om_func_generate_ulid()
  )
)
WHERE (
    jsonb_typeof(override.discounts -> 'percentage') = 'object'
    AND COALESCE(override.discounts #>> '{percentage,correlationID}', '') = ''
  ) OR (
    jsonb_typeof(override.discounts -> 'usage') = 'object'
    AND COALESCE(override.discounts #>> '{usage,correlationID}', '') = ''
  );

UPDATE charge_flat_fees charge
SET
  discounts = pg_temp.upsert_discount_correlation_id(
    charge.discounts,
    'percentage',
    COALESCE(
      (SELECT ids.percentage_correlation_id FROM charge_discount_correlation_ids ids WHERE ids.charge_id = charge.id),
      om_func_generate_ulid()
    )
  ),
  updated_at = now()
WHERE jsonb_typeof(charge.discounts -> 'percentage') = 'object'
  AND COALESCE(charge.discounts #>> '{percentage,correlationID}', '') = '';

UPDATE charge_flat_fee_overrides override
SET discounts = pg_temp.upsert_discount_correlation_id(
  override.discounts,
  'percentage',
  COALESCE(
    (SELECT ids.percentage_correlation_id FROM charge_discount_correlation_ids ids WHERE ids.charge_id = override.charge_id),
    om_func_generate_ulid()
  )
)
WHERE jsonb_typeof(override.discounts -> 'percentage') = 'object'
  AND COALESCE(override.discounts #>> '{percentage,correlationID}', '') = '';

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM charge_usage_based
    WHERE (
        jsonb_typeof(discounts -> 'percentage') = 'object'
        AND COALESCE(discounts #>> '{percentage,correlationID}', '') = ''
      ) OR (
        jsonb_typeof(discounts -> 'usage') = 'object'
        AND COALESCE(discounts #>> '{usage,correlationID}', '') = ''
      )
  ) OR EXISTS (
    SELECT 1
    FROM charge_usage_based_overrides
    WHERE (
        jsonb_typeof(discounts -> 'percentage') = 'object'
        AND COALESCE(discounts #>> '{percentage,correlationID}', '') = ''
      ) OR (
        jsonb_typeof(discounts -> 'usage') = 'object'
        AND COALESCE(discounts #>> '{usage,correlationID}', '') = ''
      )
  ) OR EXISTS (
    SELECT 1
    FROM charge_flat_fees
    WHERE jsonb_typeof(discounts -> 'percentage') = 'object'
      AND COALESCE(discounts #>> '{percentage,correlationID}', '') = ''
  ) OR EXISTS (
    SELECT 1
    FROM charge_flat_fee_overrides
    WHERE jsonb_typeof(discounts -> 'percentage') = 'object'
      AND COALESCE(discounts #>> '{percentage,correlationID}', '') = ''
  ) THEN
    RAISE EXCEPTION 'charge discounts still have missing correlation IDs after repair';
  END IF;
END
$$;

COMMIT;
