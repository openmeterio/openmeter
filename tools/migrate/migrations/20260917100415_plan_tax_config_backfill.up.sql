-- Backfill the normalized tax columns (tax_code_id, tax_behavior) from the legacy
-- JSON tax config on plan rate cards. A NULL tax_config is valid and is left
-- untouched.
--
-- The read path (productcatalog.BackfillTaxConfig) treats the normalized columns as
-- the source of truth and fills the JSON from them, but the JSON can carry a Stripe
-- code the columns cannot represent. Rows whose JSON holds a Stripe code while the
-- tax_code_id column is NULL predate the column or were never dual-written: the v3
-- tax model cannot represent them, and the next API round-trip silently deletes the
-- stored Stripe code. This migration resolves every used Stripe code to a tax code
-- entity and stamps both representations, restoring the dual-write invariant.
--
-- Provider-facing JSON fields are never removed or rewritten. Missing normalized
-- columns are copied from JSON, and missing JSON mirrors are copied from normalized
-- columns. Conflicting non-NULL representations abort instead of choosing a value.
--
-- Resolution per (namespace, Stripe code), in order:
--   1. A live tax code entity already carrying this Stripe app mapping. When more
--      than one exists, prefer system-managed, then the oldest, then the lowest ID.
--   2. No owner and the code is a Stripe code (^txcd_[0-9]{8}$): create the same
--      user-style entity as taxcode.Service.GetOrCreateByAppMapping.
--   3. Not a Stripe code: abort so the invalid tax config can be corrected.
--
-- The final checks prevent future writes from restoring the incomplete state while
-- still allowing rate cards with no tax_config, behavior-only tax configs, and
-- tax configs that contain neither a behavior nor a Stripe code.

BEGIN;

-- Existing dual-written values must already agree. Picking either side of a
-- conflict could change observable tax behavior, so leave those rows for explicit
-- operator repair.
DO $$
DECLARE
  tax_code_conflicts int;
  behavior_conflicts int;
BEGIN
  SELECT count(*) INTO tax_code_conflicts
  FROM plan_rate_cards
  WHERE tax_code_id IS NOT NULL
    AND tax_config ->> 'tax_code_id' IS NOT NULL
    AND tax_code_id::text IS DISTINCT FROM tax_config ->> 'tax_code_id';

  SELECT count(*) INTO behavior_conflicts
  FROM plan_rate_cards
  WHERE tax_behavior IS NOT NULL
    AND tax_config ->> 'behavior' IS NOT NULL
    AND tax_behavior IS DISTINCT FROM tax_config ->> 'behavior';

  IF tax_code_conflicts > 0 OR behavior_conflicts > 0 THEN
    RAISE EXCEPTION 'plan tax config backfill: % row(s) have conflicting tax_code_id representations and % row(s) have conflicting tax behavior representations; repair them before re-running', tax_code_conflicts, behavior_conflicts;
  END IF;
END $$;

-- Restore tax_code_id from the JSON's embedded tax_code_id where the column was
-- never stamped but the JSON references a live entity in the same namespace.
UPDATE plan_rate_cards r
SET tax_code_id = (r.tax_config ->> 'tax_code_id')::char(26)
WHERE r.tax_code_id IS NULL
  AND btrim(r.tax_config ->> 'tax_code_id') <> ''
  AND EXISTS (
    SELECT 1
    FROM tax_codes t
    WHERE t.id = (r.tax_config ->> 'tax_code_id')::char(26)
      AND t.namespace = r.namespace
      AND t.deleted_at IS NULL
  );

-- An embedded ID that could not be restored is invalid or references a missing,
-- deleted, or cross-namespace entity. Do not replace it from a Stripe code.
DO $$
DECLARE
  unresolved_count int;
BEGIN
  SELECT count(*) INTO unresolved_count
  FROM plan_rate_cards
  WHERE tax_code_id IS NULL
    AND tax_config ->> 'tax_code_id' IS NOT NULL;

  IF unresolved_count > 0 THEN
    RAISE EXCEPTION 'plan tax config backfill: % row(s) contain an embedded tax_code_id that cannot be resolved to a live tax code in the same namespace', unresolved_count;
  END IF;
END $$;

-- Normalized references must remain usable by the read path even when the
-- legacy JSON does not carry a provider code.
DO $$
DECLARE
  invalid_reference_count int;
BEGIN
  SELECT count(*) INTO invalid_reference_count
  FROM plan_rate_cards r
  WHERE r.tax_code_id IS NOT NULL
    AND NOT EXISTS (
      SELECT 1
      FROM tax_codes t
      WHERE t.id = r.tax_code_id
        AND t.namespace = r.namespace
        AND t.deleted_at IS NULL
    );

  IF invalid_reference_count > 0 THEN
    RAISE EXCEPTION 'plan tax config backfill: % row(s) reference a missing, deleted, or cross-namespace tax code', invalid_reference_count;
  END IF;
END $$;

-- A tax code ID and Stripe code encode the same tax identity. After restoring
-- embedded IDs, reject rows whose referenced live entity does not carry the
-- stored Stripe mapping instead of preserving two conflicting identities.
DO $$
DECLARE
  mismatched_count int;
BEGIN
  SELECT count(*) INTO mismatched_count
  FROM plan_rate_cards r
  WHERE r.tax_code_id IS NOT NULL
    AND NULLIF(btrim(r.tax_config -> 'stripe' ->> 'code'), '') IS NOT NULL
    AND NOT EXISTS (
      SELECT 1
      FROM tax_codes t,
           LATERAL jsonb_array_elements(
             CASE
               WHEN jsonb_typeof(t.app_mappings) = 'array' THEN t.app_mappings
               ELSE '[]'::jsonb
             END
           ) AS m
      WHERE t.id = r.tax_code_id
        AND t.namespace = r.namespace
        AND t.deleted_at IS NULL
        AND m ->> 'app_type' = 'stripe'
        AND m ->> 'tax_code' = btrim(r.tax_config -> 'stripe' ->> 'code')
    );

  IF mismatched_count > 0 THEN
    RAISE EXCEPTION 'plan tax config backfill: % row(s) contain a Stripe code that does not match the referenced live tax code Stripe app mapping', mismatched_count;
  END IF;
END $$;

-- Collect only the (namespace, Stripe code) pairs used by plan rate cards that
-- still need a normalized tax code reference.
DROP TABLE IF EXISTS _plan_tax_config_backfill_pairs;
CREATE TEMP TABLE _plan_tax_config_backfill_pairs AS
SELECT DISTINCT
    r.namespace,
    btrim(r.tax_config -> 'stripe' ->> 'code') AS stripe_code
FROM plan_rate_cards r
WHERE r.tax_code_id IS NULL
  AND btrim(r.tax_config -> 'stripe' ->> 'code') <> '';

CREATE INDEX ON _plan_tax_config_backfill_pairs (namespace, stripe_code);

-- Reject values the Go model would reject instead of creating entities from
-- malformed provider codes.
DO $$
DECLARE
  invalid_count int;
  invalid_codes text;
BEGIN
  SELECT count(*), string_agg(DISTINCT stripe_code, ', ')
    INTO invalid_count, invalid_codes
  FROM _plan_tax_config_backfill_pairs
  WHERE stripe_code !~ '^txcd_[0-9]{8}$';

  IF invalid_count > 0 THEN
    RAISE EXCEPTION 'plan tax config backfill: % (namespace, code) pair(s) carry a non-Stripe tax code; correct or clear these tax configs and re-run. Offending codes: %', invalid_count, invalid_codes;
  END IF;
END $$;

-- Resolve each pair to the entity selected by the read path's deterministic
-- tie-break.
DROP TABLE IF EXISTS _plan_tax_config_backfill_map;
CREATE TEMP TABLE _plan_tax_config_backfill_map (
    namespace character varying NOT NULL,
    stripe_code text NOT NULL,
    winner_id char(26) NOT NULL,
    created boolean NOT NULL DEFAULT FALSE
);

INSERT INTO _plan_tax_config_backfill_map (namespace, stripe_code, winner_id)
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
    JOIN _plan_tax_config_backfill_pairs p
      ON p.namespace = e.namespace
     AND p.stripe_code = e.app_tax_code
)
SELECT namespace, app_tax_code, id
FROM ranked
WHERE rn = 1;

-- Abort if the auto-generated key is already occupied by a live entity that does
-- not carry the matching app mapping. The service returns the same logical error.
DO $$
DECLARE
  orphaned_count int;
  orphaned text;
BEGIN
  SELECT count(*), string_agg(DISTINCT t.namespace || ':' || t.key, ', ')
    INTO orphaned_count, orphaned
  FROM _plan_tax_config_backfill_pairs p
  JOIN tax_codes t
    ON t.namespace = p.namespace
   AND t.key = 'stripe_' || p.stripe_code
   AND t.deleted_at IS NULL
  LEFT JOIN _plan_tax_config_backfill_map m
    ON m.namespace = p.namespace
   AND m.stripe_code = p.stripe_code
  WHERE m.winner_id IS NULL;

  IF orphaned_count > 0 THEN
    RAISE EXCEPTION 'plan tax config backfill: % orphaned auto-key(s); a live tax code entity holds the stripe_ key without the matching app mapping: %', orphaned_count, orphaned;
  END IF;
END $$;

-- Create a user-style tax code for pairs that found no live owner.
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
    FROM _plan_tax_config_backfill_pairs p
    WHERE NOT EXISTS (
        SELECT 1
        FROM _plan_tax_config_backfill_map m
        WHERE m.namespace = p.namespace
          AND m.stripe_code = p.stripe_code
    )
    RETURNING id, namespace, app_mappings
)
INSERT INTO _plan_tax_config_backfill_map (namespace, stripe_code, winner_id, created)
SELECT i.namespace, i.app_mappings -> 0 ->> 'tax_code', i.id, TRUE
FROM inserted i;

-- Stamp both representations together. Multiple rate cards using the same pair
-- share the one resolved tax code entity.
UPDATE plan_rate_cards r
SET tax_code_id = m.winner_id,
    tax_config = jsonb_set(
        r.tax_config,
        '{tax_code_id}',
        to_jsonb(m.winner_id::text)
    )
FROM _plan_tax_config_backfill_map m
WHERE r.namespace = m.namespace
  AND r.tax_code_id IS NULL
  AND btrim(r.tax_config -> 'stripe' ->> 'code') = m.stripe_code;

-- Mirror behavior verbatim when only the JSON representation is populated.
UPDATE plan_rate_cards
SET tax_behavior = tax_config ->> 'behavior'
WHERE tax_behavior IS NULL
  AND tax_config ->> 'behavior' IS NOT NULL;

-- Complete the reverse direction for rows written through normalized-only paths.
-- SQL NULL tax_config becomes an object only when there is tax data to preserve.
UPDATE plan_rate_cards
SET tax_config = jsonb_set(
    COALESCE(tax_config, '{}'::jsonb),
    '{tax_code_id}',
    to_jsonb(tax_code_id::text)
)
WHERE tax_code_id IS NOT NULL
  AND tax_config ->> 'tax_code_id' IS NULL;

UPDATE plan_rate_cards
SET tax_config = jsonb_set(
    COALESCE(tax_config, '{}'::jsonb),
    '{behavior}',
    to_jsonb(tax_behavior::text)
)
WHERE tax_behavior IS NOT NULL
  AND tax_config ->> 'behavior' IS NULL;

-- Fail before adding the constraints if either representation remains incomplete
-- or a Stripe code still lacks a resolved tax code.
DO $$
DECLARE
  tax_code_mismatches int;
  behavior_mismatches int;
BEGIN
  SELECT count(*) INTO tax_code_mismatches
  FROM plan_rate_cards
  WHERE tax_code_id::text IS DISTINCT FROM tax_config ->> 'tax_code_id'
     OR (
       tax_code_id IS NULL
       AND NULLIF(btrim(tax_config -> 'stripe' ->> 'code'), '') IS NOT NULL
     );

  SELECT count(*) INTO behavior_mismatches
  FROM plan_rate_cards
  WHERE tax_behavior IS DISTINCT FROM tax_config ->> 'behavior';

  IF tax_code_mismatches > 0 OR behavior_mismatches > 0 THEN
    RAISE EXCEPTION 'plan tax config backfill: plan_rate_cards still has % row(s) with inconsistent tax code representations and % row(s) with inconsistent tax behavior representations', tax_code_mismatches, behavior_mismatches;
  END IF;
END $$;

-- Prevent direct or older writers from reintroducing inconsistent representations.
-- IS NOT DISTINCT FROM makes two SQL NULL values equal, keeping tax_config optional.
ALTER TABLE "plan_rate_cards"
  ADD CONSTRAINT "plan_rate_card_tax_behavior_consistency"
    CHECK (tax_behavior IS NOT DISTINCT FROM tax_config ->> 'behavior') NOT VALID,
  ADD CONSTRAINT "plan_rate_card_tax_code_consistency"
    CHECK (
      tax_code_id::text IS NOT DISTINCT FROM tax_config ->> 'tax_code_id'
      AND (
        NULLIF(btrim(tax_config -> 'stripe' ->> 'code'), '') IS NULL
        OR tax_code_id IS NOT NULL
      )
    ) NOT VALID;

DO $$
DECLARE
  attached int;
  created_count int;
BEGIN
  SELECT count(*) INTO attached FROM _plan_tax_config_backfill_map WHERE NOT created;
  SELECT count(*) INTO created_count FROM _plan_tax_config_backfill_map WHERE created;
  RAISE NOTICE 'plan tax config backfill: resolved % (namespace, code) pair(s); % attached to existing tax codes, % created new tax codes', attached + created_count, attached, created_count;
END $$;

-- Session-scoped temp tables are dropped explicitly because Atlas dry-run
-- validation cannot model ON COMMIT DROP.
DROP TABLE _plan_tax_config_backfill_pairs;
DROP TABLE _plan_tax_config_backfill_map;

COMMIT;
