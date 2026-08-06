-- Fail before changing data when a legacy status cannot be derived from durable realizations.
DO $$
DECLARE
  missing_credit_grant_count bigint;
  unsupported_payment_status_count bigint;
BEGIN
  SELECT COUNT(*)
  INTO missing_credit_grant_count
  FROM "charge_credit_purchases" AS "credit_purchase"
  WHERE "credit_purchase"."status_detailed" = 'active'
    AND "credit_purchase"."settlement" ->> 'type' = 'invoice'
    AND NOT EXISTS (
      SELECT 1
      FROM "charge_credit_purchase_credit_grants" AS "credit_grant"
      WHERE "credit_grant"."charge_id" = "credit_purchase"."id"
        AND "credit_grant"."deleted_at" IS NULL
        AND "credit_grant"."transaction_group_id" <> ''
    );

  IF missing_credit_grant_count > 0 THEN
    RAISE EXCEPTION 'migrate_invoice_credit_purchase_statuses: % active invoice credit purchase(s) have no active credit grant', missing_credit_grant_count;
  END IF;

  SELECT COUNT(*)
  INTO unsupported_payment_status_count
  FROM "charge_credit_purchases" AS "credit_purchase"
  JOIN "charge_credit_purchase_invoiced_payments" AS "invoiced_payment"
    ON "invoiced_payment"."charge_id" = "credit_purchase"."id"
    AND "invoiced_payment"."deleted_at" IS NULL
  WHERE "credit_purchase"."status_detailed" = 'active'
    AND "credit_purchase"."settlement" ->> 'type' = 'invoice'
    AND "invoiced_payment"."status" NOT IN ('authorized', 'settled');

  IF unsupported_payment_status_count > 0 THEN
    RAISE EXCEPTION 'migrate_invoice_credit_purchase_statuses: % active invoice credit purchase(s) have an unsupported invoiced payment status', unsupported_payment_status_count;
  END IF;
END
$$;

-- Replace the legacy generic status with the state derived from the payment realization.
WITH "mapped_credit_purchases" AS (
  SELECT
    "credit_purchase"."id",
    CASE "invoiced_payment"."status"
      WHEN 'authorized' THEN 'active.payment.authorized'
      WHEN 'settled' THEN 'final'
      ELSE 'active.payment.pending'
    END AS "status_detailed",
    CASE "invoiced_payment"."status"
      WHEN 'settled' THEN 'final'
      ELSE 'active'
    END AS "status"
  FROM "charge_credit_purchases" AS "credit_purchase"
  LEFT JOIN "charge_credit_purchase_invoiced_payments" AS "invoiced_payment"
    ON "invoiced_payment"."charge_id" = "credit_purchase"."id"
    AND "invoiced_payment"."deleted_at" IS NULL
  WHERE "credit_purchase"."status_detailed" = 'active'
    AND "credit_purchase"."settlement" ->> 'type' = 'invoice'
)
UPDATE "charge_credit_purchases" AS "credit_purchase"
SET
  "status_detailed" = "mapped_credit_purchases"."status_detailed",
  "status" = "mapped_credit_purchases"."status"
FROM "mapped_credit_purchases"
WHERE "credit_purchase"."id" = "mapped_credit_purchases"."id";
