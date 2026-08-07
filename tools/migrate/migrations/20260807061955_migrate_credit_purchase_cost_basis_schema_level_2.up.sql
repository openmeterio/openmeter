BEGIN;

-- The application uses the same row lock when lazily upgrading a legacy charge.
-- Locking in ID order makes the two upgrade paths serialize deterministically.
DO $migration$
BEGIN
  PERFORM "id"
  FROM "charge_credit_purchases"
  WHERE "schema_level" = 1
  ORDER BY "id"
  FOR UPDATE;
END
$migration$;

DO $migration$
DECLARE
  invalid_ids TEXT;
BEGIN
  SELECT string_agg("id"::text, ', ' ORDER BY "id")
  INTO invalid_ids
  FROM "charge_credit_purchases"
  WHERE "schema_level" = 1
    AND COALESCE("settlement"->>'type', '') NOT IN ('invoice', 'external', 'promotional');

  IF invalid_ids IS NOT NULL THEN
    RAISE EXCEPTION 'cannot migrate credit purchases with unsupported settlement types [ids=%]', invalid_ids;
  END IF;

  SELECT string_agg("id"::text, ', ' ORDER BY "id")
  INTO invalid_ids
  FROM "charge_credit_purchases"
  WHERE "schema_level" = 1
    AND (
      "fiat_cost_basis" IS NOT NULL
      OR "cost_basis_id" IS NOT NULL
      OR "settlement_type" IS NOT NULL
      OR "initial_payment_settlement_status" IS NOT NULL
    );

  IF invalid_ids IS NOT NULL THEN
    RAISE EXCEPTION 'cannot migrate legacy credit purchases with dedicated schema-level state [ids=%]', invalid_ids;
  END IF;

  SELECT string_agg("id"::text, ', ' ORDER BY "id")
  INTO invalid_ids
  FROM "charge_credit_purchases"
  WHERE "schema_level" = 1
    AND "settlement"->>'type' IN ('invoice', 'external')
    AND CASE
      WHEN "settlement"->>'costBasis' IS NULL THEN TRUE
      WHEN "settlement"->>'costBasis' !~ '^[0-9]+(\.[0-9]+)?$' THEN TRUE
      ELSE ("settlement"->>'costBasis')::numeric <= 0
    END;

  IF invalid_ids IS NOT NULL THEN
    RAISE EXCEPTION 'cannot migrate credit purchases with an invalid cost basis [ids=%]', invalid_ids;
  END IF;

  SELECT string_agg("id"::text, ', ' ORDER BY "id")
  INTO invalid_ids
  FROM "charge_credit_purchases"
  WHERE "schema_level" = 1
    AND "settlement"->>'type' IN ('invoice', 'external')
    AND (
      COALESCE("settlement"->>'currency', '') !~ '^[A-Z]{3}$'
      OR ("currency" IS NULL) = ("custom_currency_id" IS NULL)
      OR (
        "custom_currency_id" IS NULL
        AND "settlement"->>'currency' IS DISTINCT FROM "currency"
      )
    );

  IF invalid_ids IS NOT NULL THEN
    RAISE EXCEPTION 'cannot migrate credit purchases with an invalid settlement currency [ids=%]', invalid_ids;
  END IF;

  SELECT string_agg("id"::text, ', ' ORDER BY "id")
  INTO invalid_ids
  FROM "charge_credit_purchases"
  WHERE "schema_level" = 1
    AND (
      (
        "settlement"->>'type' = 'external'
        AND COALESCE("settlement"->>'status', '') NOT IN ('created', 'authorized', 'settled')
      )
      OR (
        "settlement"->>'type' = 'invoice'
        AND "settlement" ? 'status'
      )
      OR (
        "settlement"->>'type' = 'promotional'
        AND (
          "settlement" ? 'currency'
          OR "settlement" ? 'costBasis'
          OR "settlement" ? 'status'
        )
      )
    );

  IF invalid_ids IS NOT NULL THEN
    RAISE EXCEPTION 'cannot migrate credit purchases with invalid settlement compatibility fields [ids=%]', invalid_ids;
  END IF;
END
$migration$;

CREATE TEMP TABLE "om_credit_purchase_cost_basis_backfill" (
  "charge_id" text PRIMARY KEY,
  "cost_basis_id" character(26) NOT NULL UNIQUE
) ON COMMIT DROP;

INSERT INTO "om_credit_purchase_cost_basis_backfill" ("charge_id", "cost_basis_id")
SELECT "id", om_func_generate_ulid()
FROM "charge_credit_purchases"
WHERE "schema_level" = 1
  AND "settlement"->>'type' IN ('invoice', 'external')
  AND "custom_currency_id" IS NOT NULL
ORDER BY "id";

INSERT INTO "charge_credit_purchase_cost_bases" (
  "id",
  "mode",
  "fiat_currency",
  "manual_rate",
  "resolved_cost_basis",
  "resolved_at",
  "namespace",
  "created_at",
  "updated_at",
  "deleted_at",
  "currency_cost_basis_id",
  "resolved_cost_basis_id",
  "currency_id"
)
SELECT
  "backfill"."cost_basis_id",
  'manual',
  "credit_purchase"."settlement"->>'currency',
  ("credit_purchase"."settlement"->>'costBasis')::numeric,
  ("credit_purchase"."settlement"->>'costBasis')::numeric,
  "credit_purchase"."created_at",
  "credit_purchase"."namespace",
  CURRENT_TIMESTAMP,
  CURRENT_TIMESTAMP,
  NULL,
  NULL,
  NULL,
  "credit_purchase"."custom_currency_id"
FROM "om_credit_purchase_cost_basis_backfill" AS "backfill"
JOIN "charge_credit_purchases" AS "credit_purchase"
  ON "credit_purchase"."id" = "backfill"."charge_id";

UPDATE "charge_credit_purchases" AS "credit_purchase"
SET "cost_basis_id" = "backfill"."cost_basis_id"
FROM "om_credit_purchase_cost_basis_backfill" AS "backfill"
WHERE "credit_purchase"."id" = "backfill"."charge_id"
  AND "credit_purchase"."schema_level" = 1;

-- schema_level is the commit marker and is set only after the authoritative
-- cost-basis representation has been populated.
UPDATE "charge_credit_purchases"
SET
  "fiat_cost_basis" = CASE
    WHEN "settlement"->>'type' IN ('invoice', 'external')
      AND "custom_currency_id" IS NULL
    THEN ("settlement"->>'costBasis')::numeric
    ELSE NULL
  END,
  "settlement_type" = "settlement"->>'type',
  "initial_payment_settlement_status" = CASE
    WHEN "settlement"->>'type' = 'external' THEN "settlement"->>'status'
    ELSE NULL
  END,
  "schema_level" = 2
WHERE "schema_level" = 1;

DO $migration$
DECLARE
  invalid_ids TEXT;
BEGIN
  SELECT string_agg("id"::text, ', ' ORDER BY "id")
  INTO invalid_ids
  FROM "charge_credit_purchases"
  WHERE "schema_level" <> 2;

  IF invalid_ids IS NOT NULL THEN
    RAISE EXCEPTION 'credit purchase cost-basis migration left rows below schema level 2 [ids=%]', invalid_ids;
  END IF;

  SELECT string_agg("id"::text, ', ' ORDER BY "id")
  INTO invalid_ids
  FROM "charge_credit_purchases"
  WHERE "settlement_type" IS DISTINCT FROM "settlement"->>'type'
    OR "initial_payment_settlement_status" IS DISTINCT FROM CASE
      WHEN "settlement"->>'type' = 'external' THEN "settlement"->>'status'
      ELSE NULL
    END;

  IF invalid_ids IS NOT NULL THEN
    RAISE EXCEPTION 'credit purchase settlement state does not match its compatibility shadow [ids=%]', invalid_ids;
  END IF;

  SELECT string_agg("id"::text, ', ' ORDER BY "id")
  INTO invalid_ids
  FROM "charge_credit_purchases"
  WHERE (
      "settlement_type" = 'promotional'
      AND ("fiat_cost_basis" IS NOT NULL OR "cost_basis_id" IS NOT NULL)
    )
    OR (
      "settlement_type" IN ('invoice', 'external')
      AND "custom_currency_id" IS NULL
      AND (
        "fiat_cost_basis" IS DISTINCT FROM ("settlement"->>'costBasis')::numeric
        OR "cost_basis_id" IS NOT NULL
        OR "currency" IS DISTINCT FROM "settlement"->>'currency'
      )
    );

  IF invalid_ids IS NOT NULL THEN
    RAISE EXCEPTION 'credit purchase fiat cost basis does not match its compatibility shadow [ids=%]', invalid_ids;
  END IF;

  SELECT string_agg("credit_purchase"."id"::text, ', ' ORDER BY "credit_purchase"."id")
  INTO invalid_ids
  FROM "charge_credit_purchases" AS "credit_purchase"
  LEFT JOIN "charge_credit_purchase_cost_bases" AS "cost_basis"
    ON "cost_basis"."id" = "credit_purchase"."cost_basis_id"
  WHERE "credit_purchase"."settlement_type" IN ('invoice', 'external')
    AND "credit_purchase"."custom_currency_id" IS NOT NULL
    AND (
      "credit_purchase"."fiat_cost_basis" IS NOT NULL
      OR "credit_purchase"."cost_basis_id" IS NULL
      OR "cost_basis"."id" IS NULL
      OR "cost_basis"."namespace" IS DISTINCT FROM "credit_purchase"."namespace"
      OR "cost_basis"."currency_id" IS DISTINCT FROM "credit_purchase"."custom_currency_id"
      OR "cost_basis"."fiat_currency" IS DISTINCT FROM "credit_purchase"."settlement"->>'currency'
      OR "cost_basis"."resolved_cost_basis" IS DISTINCT FROM ("credit_purchase"."settlement"->>'costBasis')::numeric
    );

  IF invalid_ids IS NOT NULL THEN
    RAISE EXCEPTION 'credit purchase custom-currency cost basis does not match its compatibility shadow [ids=%]', invalid_ids;
  END IF;

  SELECT string_agg("credit_purchase"."id"::text, ', ' ORDER BY "credit_purchase"."id")
  INTO invalid_ids
  FROM "om_credit_purchase_cost_basis_backfill" AS "backfill"
  JOIN "charge_credit_purchases" AS "credit_purchase"
    ON "credit_purchase"."id" = "backfill"."charge_id"
  JOIN "charge_credit_purchase_cost_bases" AS "cost_basis"
    ON "cost_basis"."id" = "backfill"."cost_basis_id"
  WHERE "cost_basis"."mode" <> 'manual'
    OR "cost_basis"."manual_rate" IS DISTINCT FROM ("credit_purchase"."settlement"->>'costBasis')::numeric
    OR "cost_basis"."resolved_at" IS DISTINCT FROM "credit_purchase"."created_at"
    OR "cost_basis"."currency_cost_basis_id" IS NOT NULL
    OR "cost_basis"."resolved_cost_basis_id" IS NOT NULL;

  IF invalid_ids IS NOT NULL THEN
    RAISE EXCEPTION 'migrated custom-currency cost basis is not a resolved manual cost basis [ids=%]', invalid_ids;
  END IF;
END
$migration$;

COMMIT;
