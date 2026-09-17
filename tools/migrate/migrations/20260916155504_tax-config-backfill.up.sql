-- Backfill the normalized tax columns (tax_code_id, tax_behavior) from the legacy
-- JSON tax configs. Part 1: billing_workflow_configs, the billing profile default
-- tax configuration. Succeeding parts extend the pair collection and the stamps with
-- plan_rate_cards, addon_rate_cards, subscription_items and billing_invoice_lines.
--
-- The read path (productcatalog.BackfillTaxConfig) treats the normalized columns as
-- the source of truth and fills the JSON from them, but the JSON can carry a Stripe
-- code the columns cannot represent. Rows whose JSON holds a stripe code while the
-- tax_code_id column is NULL predate the column or were never dual-written: the v3
-- tax model cannot represent them, and the next API round-trip silently deletes the
-- stored stripe code. This migration resolves every used stripe code to a tax code
-- entity and stamps both representations, restoring the dual-write invariant.
--
-- The legacy JSON values are never removed and never rewritten beyond adding the
-- embedded tax_code_id: invoice calculation reads the JSON, so the migration is
-- representation-only and cannot change tax outcomes.
--
-- Resolution per (namespace, stripe code), in order:
--   1. A live tax code entity already carrying this stripe app mapping. This finds
--      the seeded template entities (saas_business, paas, ...) and everything the
--      GetOrCreateByAppMapping path created, so creation stays the exception.
--   2. No owner and the code is a Stripe code (^txcd_[0-9]{8}$, the same pattern the
--      Go validation enforces): create a user-style entity exactly like the Go
--      auto-create path does (key and name derived from the code, no annotations).
--   3. Not a Stripe code: abort and list the offending codes. These rows carry
--      garbage a migration must not guess about.
--
-- The migration is idempotent: every statement matches on the NULL state it fixes.

BEGIN;

-- Step 1: restore tax_code_id from the JSON's own embedded tax_code_id where the
-- column was never stamped but the JSON references a live tax code entity in the
-- namespace. Both representations are normally written together, so this is
-- expected to be a no-op; it exists so the final guard only fires on genuinely
-- unresolvable data.
UPDATE billing_workflow_configs w
SET tax_code_id = (w.invoice_default_tax_settings ->> 'tax_code_id')::char(26)
WHERE w.tax_code_id IS NULL
  AND btrim(w.invoice_default_tax_settings ->> 'tax_code_id') <> ''
  AND EXISTS (
    SELECT 1
    FROM tax_codes t
    WHERE t.id = (w.invoice_default_tax_settings ->> 'tax_code_id')::char(26)
      AND t.namespace = w.namespace
      AND t.deleted_at IS NULL
  );

-- Step 2: collect the (namespace, stripe_code) pairs that need resolution — the
-- codes actually used by the data, nothing else.
DROP TABLE IF EXISTS _tax_config_backfill_pairs;
CREATE TEMP TABLE _tax_config_backfill_pairs AS
SELECT DISTINCT
    w.namespace,
    btrim(w.invoice_default_tax_settings -> 'stripe' ->> 'code') AS stripe_code
FROM billing_workflow_configs w
WHERE w.tax_code_id IS NULL
  AND btrim(w.invoice_default_tax_settings -> 'stripe' ->> 'code') <> '';

CREATE INDEX ON _tax_config_backfill_pairs (namespace, stripe_code);

-- Step 3: fail loudly on codes that are not Stripe tax codes at all (the Go
-- validation enforces the same pattern, taxcode.TaxCodeStripeRegexp).
DO $$
DECLARE
  invalid_count int;
  invalid_codes text;
BEGIN
  SELECT count(*), string_agg(DISTINCT stripe_code, ', ')
    INTO invalid_count, invalid_codes
  FROM _tax_config_backfill_pairs
  WHERE stripe_code !~ '^txcd_[0-9]{8}$';

  IF invalid_count > 0 THEN
    RAISE EXCEPTION 'tax config backfill: % (namespace, code) pair(s) carry a non-Stripe tax code; correct or clear these tax configs and re-run. Offending codes: %', invalid_count, invalid_codes;
  END IF;
END $$;

-- Step 4: resolve each pair to a live tax code entity that already carries this
-- stripe app mapping. When several entities carry the same mapping, pick what the
-- read side picks: system-managed first, then the oldest.
DROP TABLE IF EXISTS _tax_config_backfill_map;
CREATE TEMP TABLE _tax_config_backfill_map (
    namespace character varying NOT NULL,
    stripe_code text NOT NULL,
    winner_id char(26) NOT NULL,
    created boolean NOT NULL DEFAULT FALSE
);

INSERT INTO _tax_config_backfill_map (namespace, stripe_code, winner_id)
WITH expanded AS (
    SELECT
        t.id,
        t.namespace,
        t.created_at,
        m ->> 'tax_code' AS app_tax_code,
        COALESCE(t.annotations ->> 'managed_by', '') = 'system' AS is_system
    FROM tax_codes t,
         LATERAL jsonb_array_elements(COALESCE(t.app_mappings, '[]'::jsonb)) AS m
    WHERE t.deleted_at IS NULL
      AND m ->> 'app_type' = 'stripe'
),
ranked AS (
    SELECT
        e.id,
        e.namespace,
        e.app_tax_code,
        ROW_NUMBER() OVER (
            PARTITION BY e.namespace, e.app_tax_code
            ORDER BY e.is_system DESC, e.created_at ASC, e.id ASC
        ) AS rn
    FROM expanded e
    JOIN _tax_config_backfill_pairs p
      ON p.namespace = e.namespace
     AND p.stripe_code = e.app_tax_code
)
SELECT namespace, app_tax_code, id
FROM ranked
WHERE rn = 1;

-- Step 5: fail loudly when a live tax code entity already holds the auto-generated
-- key ('stripe_' || code) without carrying the app mapping — the INSERT below
-- would collide with the unique (namespace, key) index. This mirrors the
-- orphaned-key error the Go GetOrCreateByAppMapping path returns; a human must
-- decide what these keys mean.
DO $$
DECLARE
  orphaned_count int;
  orphaned text;
BEGIN
  SELECT count(*), string_agg(DISTINCT t.namespace || ':' || t.key, ', ')
    INTO orphaned_count, orphaned
  FROM _tax_config_backfill_pairs p
  JOIN tax_codes t
    ON t.namespace = p.namespace
   AND t.key = 'stripe_' || p.stripe_code
   AND t.deleted_at IS NULL
  LEFT JOIN _tax_config_backfill_map m
    ON m.namespace = p.namespace
   AND m.stripe_code = p.stripe_code
  WHERE m.winner_id IS NULL;

  IF orphaned_count > 0 THEN
    RAISE EXCEPTION 'tax config backfill: % orphaned auto-key(s); a live tax code entity holds the stripe_ key without the matching app mapping: %', orphaned_count, orphaned;
  END IF;
END $$;

-- Step 6: create a user-style tax code entity for pairs that found no owner —
-- exactly what the Go auto-create path (taxcode Service.GetOrCreateByAppMapping)
-- produces: key and name derived from the stripe code, no annotations. The ULID
-- generator comes from migration 20250807075408 (same reliance as 20250818093933).
WITH inserted AS (
    INSERT INTO tax_codes (id, namespace, created_at, updated_at, name, key, app_mappings)
    SELECT
        om_func_generate_ulid(),
        p.namespace,
        NOW(),
        NOW(),
        p.stripe_code,
        'stripe_' || p.stripe_code,
        jsonb_build_array(jsonb_build_object('app_type', 'stripe', 'tax_code', p.stripe_code))
    FROM _tax_config_backfill_pairs p
    WHERE NOT EXISTS (
        SELECT 1
        FROM _tax_config_backfill_map m
        WHERE m.namespace = p.namespace
          AND m.stripe_code = p.stripe_code
    )
    RETURNING id, namespace, app_mappings
)
INSERT INTO _tax_config_backfill_map (namespace, stripe_code, winner_id, created)
SELECT i.namespace, i.app_mappings -> 0 ->> 'tax_code', i.id, TRUE
FROM inserted i;

-- Step 7: stamp the resolved tax code onto billing_workflow_configs. Both
-- representations are updated together — the column is what the application
-- reads, the embedded JSON tax_code_id is what raw consumers see.
UPDATE billing_workflow_configs w
SET tax_code_id = m.winner_id,
    invoice_default_tax_settings = jsonb_set(
        w.invoice_default_tax_settings,
        '{tax_code_id}',
        to_jsonb(m.winner_id::text)
    )
FROM _tax_config_backfill_map m
WHERE w.namespace = m.namespace
  AND w.tax_code_id IS NULL
  AND btrim(w.invoice_default_tax_settings -> 'stripe' ->> 'code') = m.stripe_code;

-- Step 8: stamp tax_behavior from the JSON behavior where the column is NULL.
-- The value is mirrored verbatim; it is a provider-facing enum string, not a FK.
UPDATE billing_workflow_configs
SET tax_behavior = invoice_default_tax_settings ->> 'behavior'
WHERE tax_behavior IS NULL
  AND btrim(invoice_default_tax_settings ->> 'behavior') <> '';

-- Step 9: fail loudly if anything is left. After this migration no workflow
-- config may carry a stripe code without a tax code FK, nor a behavior without
-- the normalized column.
DO $$
DECLARE
  stripe_missing int;
  behavior_missing int;
BEGIN
  SELECT count(*) INTO stripe_missing
  FROM billing_workflow_configs
  WHERE tax_code_id IS NULL
    AND btrim(invoice_default_tax_settings -> 'stripe' ->> 'code') <> '';

  SELECT count(*) INTO behavior_missing
  FROM billing_workflow_configs
  WHERE tax_behavior IS NULL
    AND btrim(invoice_default_tax_settings ->> 'behavior') <> '';

  IF stripe_missing > 0 OR behavior_missing > 0 THEN
    RAISE EXCEPTION 'tax config backfill: billing_workflow_configs still has % row(s) with a stripe code but no tax_code_id and % row(s) with a behavior but no tax_behavior column value', stripe_missing, behavior_missing;
  END IF;
END $$;

-- Step 10: audit trail — how many codes attached to existing entities versus
-- created new ones, so the migration log answers review questions.
DO $$
DECLARE
  attached int;
  created_count int;
BEGIN
  SELECT count(*) INTO attached FROM _tax_config_backfill_map WHERE NOT created;
  SELECT count(*) INTO created_count FROM _tax_config_backfill_map WHERE created;
  RAISE NOTICE 'tax config backfill: resolved % (namespace, code) pair(s); % attached to existing tax codes, % created new tax codes', attached + created_count, attached, created_count;
END $$;

-- Session-scoped temp tables are dropped explicitly: ON COMMIT DROP cannot be
-- modeled by the atlas dry-run validation (see the dedupe migration).
DROP TABLE _tax_config_backfill_pairs;
DROP TABLE _tax_config_backfill_map;

COMMIT;
