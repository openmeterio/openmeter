-- Repair invoiced credit-purchase payments that stored the purchased credit amount
-- instead of the final fiat total of their standard invoice line.
BEGIN;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM charge_credit_purchase_invoiced_payments p
    LEFT JOIN billing_invoice_lines l
      ON l.id = p.line_id
    WHERE l.id IS NULL
      OR l.namespace IS DISTINCT FROM p.namespace
      OR l.invoice_id IS DISTINCT FROM p.invoice_id
  ) THEN
    RAISE EXCEPTION 'cannot backfill credit-purchase invoice payments with mismatched invoice-line references';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM charge_credit_purchase_invoiced_payments p
    JOIN billing_invoice_lines l
      ON l.id = p.line_id
     AND l.namespace = p.namespace
     AND l.invoice_id = p.invoice_id
    WHERE l.total <= 0
  ) THEN
    RAISE EXCEPTION 'cannot backfill credit-purchase invoice payments from non-positive invoice-line totals';
  END IF;
END
$$;

UPDATE charge_credit_purchase_invoiced_payments p
SET
  amount = l.total,
  updated_at = now()
FROM billing_invoice_lines l
WHERE l.id = p.line_id
  AND l.namespace = p.namespace
  AND l.invoice_id = p.invoice_id
  AND p.amount IS DISTINCT FROM l.total;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM charge_credit_purchase_invoiced_payments p
    JOIN billing_invoice_lines l
      ON l.id = p.line_id
     AND l.namespace = p.namespace
     AND l.invoice_id = p.invoice_id
    WHERE p.amount IS DISTINCT FROM l.total
  ) THEN
    RAISE EXCEPTION 'credit-purchase invoice payment fiat amount backfill is incomplete';
  END IF;
END
$$;

COMMIT;
