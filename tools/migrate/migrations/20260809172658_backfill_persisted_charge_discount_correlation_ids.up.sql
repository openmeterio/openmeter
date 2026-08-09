-- Repair persisted charge intent discounts missed by the gathering-line backfill.
-- Existing IDs are preserved; missing IDs are generated independently.
BEGIN;

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

UPDATE charge_usage_based
SET
  discounts = pg_temp.upsert_discount_correlation_id(
    pg_temp.upsert_discount_correlation_id(discounts, 'percentage', om_func_generate_ulid()),
    'usage',
    om_func_generate_ulid()
  ),
  updated_at = now()
WHERE (
  jsonb_typeof(discounts -> 'percentage') = 'object'
  AND COALESCE(discounts #>> '{percentage,correlationID}', '') = ''
) OR (
  jsonb_typeof(discounts -> 'usage') = 'object'
  AND COALESCE(discounts #>> '{usage,correlationID}', '') = ''
);

UPDATE charge_usage_based_overrides
SET discounts = pg_temp.upsert_discount_correlation_id(
  pg_temp.upsert_discount_correlation_id(discounts, 'percentage', om_func_generate_ulid()),
  'usage',
  om_func_generate_ulid()
)
WHERE (
  jsonb_typeof(discounts -> 'percentage') = 'object'
  AND COALESCE(discounts #>> '{percentage,correlationID}', '') = ''
) OR (
  jsonb_typeof(discounts -> 'usage') = 'object'
  AND COALESCE(discounts #>> '{usage,correlationID}', '') = ''
);

UPDATE charge_flat_fees
SET
  discounts = pg_temp.upsert_discount_correlation_id(
    pg_temp.upsert_discount_correlation_id(discounts, 'percentage', om_func_generate_ulid()),
    'usage',
    om_func_generate_ulid()
  ),
  updated_at = now()
WHERE (
  jsonb_typeof(discounts -> 'percentage') = 'object'
  AND COALESCE(discounts #>> '{percentage,correlationID}', '') = ''
) OR (
  jsonb_typeof(discounts -> 'usage') = 'object'
  AND COALESCE(discounts #>> '{usage,correlationID}', '') = ''
);

UPDATE charge_flat_fee_overrides
SET discounts = pg_temp.upsert_discount_correlation_id(
  pg_temp.upsert_discount_correlation_id(discounts, 'percentage', om_func_generate_ulid()),
  'usage',
  om_func_generate_ulid()
)
WHERE (
  jsonb_typeof(discounts -> 'percentage') = 'object'
  AND COALESCE(discounts #>> '{percentage,correlationID}', '') = ''
) OR (
  jsonb_typeof(discounts -> 'usage') = 'object'
  AND COALESCE(discounts #>> '{usage,correlationID}', '') = ''
);

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM (
      SELECT discounts FROM charge_usage_based
      UNION ALL
      SELECT discounts FROM charge_usage_based_overrides
      UNION ALL
      SELECT discounts FROM charge_flat_fees
      UNION ALL
      SELECT discounts FROM charge_flat_fee_overrides
    ) persisted_charge_discounts
    WHERE (
      jsonb_typeof(discounts -> 'percentage') = 'object'
      AND COALESCE(discounts #>> '{percentage,correlationID}', '') = ''
    ) OR (
      jsonb_typeof(discounts -> 'usage') = 'object'
      AND COALESCE(discounts #>> '{usage,correlationID}', '') = ''
    )
  ) THEN
    RAISE EXCEPTION 'persisted charge discounts still have missing correlation IDs after repair';
  END IF;
END
$$;

COMMIT;
