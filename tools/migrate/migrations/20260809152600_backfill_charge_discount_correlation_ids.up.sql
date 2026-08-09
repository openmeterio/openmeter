-- Repair charge-backed gathering-line discount snapshots so detailed-line rating can rely on
-- correlation IDs. Existing IDs are preserved; the generated value has no charge semantics.
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

UPDATE billing_gathering_invoice_lines line
SET
  ratecard_discounts = pg_temp.upsert_discount_correlation_id(
    pg_temp.upsert_discount_correlation_id(
      line.ratecard_discounts,
      'percentage',
      om_func_generate_ulid()
    ),
    'usage',
    om_func_generate_ulid()
  ),
  updated_at = now()
WHERE line.charge_id IS NOT NULL
  AND line.deleted_at IS NULL
  AND (
    (
      jsonb_typeof(line.ratecard_discounts -> 'percentage') = 'object'
      AND COALESCE(line.ratecard_discounts #>> '{percentage,correlationID}', '') = ''
    ) OR (
      jsonb_typeof(line.ratecard_discounts -> 'usage') = 'object'
      AND COALESCE(line.ratecard_discounts #>> '{usage,correlationID}', '') = ''
    )
  );

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM billing_gathering_invoice_lines line
    WHERE line.charge_id IS NOT NULL
      AND line.deleted_at IS NULL
      AND (
        (
          jsonb_typeof(line.ratecard_discounts -> 'percentage') = 'object'
          AND COALESCE(line.ratecard_discounts #>> '{percentage,correlationID}', '') = ''
        ) OR (
          jsonb_typeof(line.ratecard_discounts -> 'usage') = 'object'
          AND COALESCE(line.ratecard_discounts #>> '{usage,correlationID}', '') = ''
        )
      )
  ) THEN
    RAISE EXCEPTION 'charge-backed gathering-line discounts still have missing correlation IDs after repair';
  END IF;
END
$$;

COMMIT;
