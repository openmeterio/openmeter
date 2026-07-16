-- Remove usage discounts from flat-priced invoice lines. Usage discounts cannot be applied to
-- flat prices, but the legacy pending-line validator used to skip usage validation whenever a
-- percentage discount was also present.
BEGIN;

UPDATE billing_invoice_lines l
SET
  ratecard_discounts = NULLIF(l.ratecard_discounts - 'usage', '{}'::jsonb),
  updated_at = now()
FROM billing_invoice_usage_based_line_configs u
WHERE u.namespace = l.namespace
  AND u.id = l.usage_based_line_config_id
  AND u.price_type = 'flat'
  AND l.ratecard_discounts ? 'usage';

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM billing_invoice_lines l
    JOIN billing_invoice_usage_based_line_configs u
      ON u.namespace = l.namespace
     AND u.id = l.usage_based_line_config_id
    WHERE u.price_type = 'flat'
      AND l.ratecard_discounts ? 'usage'
  ) THEN
    RAISE EXCEPTION 'flat-price invoice lines still contain usage discounts after repair';
  END IF;
END
$$;

COMMIT;
