-- Backfill normalized tax columns from legacy JSON for rows belonging to
-- standard invoices. The preceding migration repaired gathering rows; this
-- migration completes the historical standard-line population before common
-- table-wide consistency checks are installed.
--
-- Provider-facing JSON fields are preserved. Missing normalized columns are
-- copied from JSON, missing JSON mirrors are copied from normalized columns,
-- and conflicting identities abort instead of choosing a value.

BEGIN;

-- Snapshot the rows in scope so every statement operates on the same standard
-- line population while invoice lifecycle updates continue.
CREATE TEMP TABLE _standard_invoice_line_tax_scope AS
SELECT l.id, l.namespace, l.invoice_id
FROM billing_invoice_lines l
JOIN billing_invoices i
  ON i.id = l.invoice_id
 AND i.namespace = l.namespace
WHERE i.status <> 'gathering';

CREATE UNIQUE INDEX ON _standard_invoice_line_tax_scope (id);

DO $$
DECLARE
  tax_code_conflicts int;
  behavior_conflicts int;
  invalid_behavior_count int;
BEGIN
  SELECT count(*) INTO tax_code_conflicts
  FROM billing_invoice_lines r
  JOIN _standard_invoice_line_tax_scope s ON s.id = r.id AND s.invoice_id = r.invoice_id
  WHERE r.tax_code_id IS NOT NULL
    AND r.tax_config ->> 'tax_code_id' IS NOT NULL
    AND r.tax_code_id::text IS DISTINCT FROM r.tax_config ->> 'tax_code_id';

  SELECT count(*) INTO behavior_conflicts
  FROM billing_invoice_lines r
  JOIN _standard_invoice_line_tax_scope s ON s.id = r.id AND s.invoice_id = r.invoice_id
  WHERE r.tax_behavior IS NOT NULL
    AND r.tax_config ->> 'behavior' IS NOT NULL
    AND r.tax_behavior IS DISTINCT FROM r.tax_config ->> 'behavior';

  SELECT count(*) INTO invalid_behavior_count
  FROM billing_invoice_lines r
  JOIN _standard_invoice_line_tax_scope s ON s.id = r.id AND s.invoice_id = r.invoice_id
  WHERE (
      r.tax_behavior IS NOT NULL
      AND r.tax_behavior NOT IN ('inclusive', 'exclusive')
    ) OR (
      r.tax_config ->> 'behavior' IS NOT NULL
      AND r.tax_config ->> 'behavior' NOT IN ('inclusive', 'exclusive')
    );

  IF tax_code_conflicts > 0 OR behavior_conflicts > 0 THEN
    RAISE EXCEPTION 'standard invoice line tax config backfill: % row(s) have conflicting tax_code_id representations and % row(s) have conflicting tax behavior representations; repair them before re-running', tax_code_conflicts, behavior_conflicts;
  END IF;

  IF invalid_behavior_count > 0 THEN
    RAISE EXCEPTION 'standard invoice line tax config backfill: % row(s) contain invalid tax behavior; only inclusive or exclusive are allowed', invalid_behavior_count;
  END IF;
END $$;

-- Tax-code app mappings participate in identity resolution but cannot be
-- protected by an invoice-line CHECK constraint. Block mapping writers for the
-- remainder of the repair so selection, creation, and validation use one stable
-- identity set. Reads remain available.
LOCK TABLE tax_codes IN SHARE ROW EXCLUSIVE MODE;

-- Restore references embedded in historical JSON when the entity still exists
-- in the same namespace. Soft-deleted tax codes remain valid invoice history.
UPDATE billing_invoice_lines r
SET tax_code_id = (r.tax_config ->> 'tax_code_id')::char(26)
FROM _standard_invoice_line_tax_scope s
WHERE s.id = r.id
  AND s.invoice_id = r.invoice_id
  AND r.tax_code_id IS NULL
  AND btrim(r.tax_config ->> 'tax_code_id') <> ''
  AND EXISTS (
    SELECT 1
    FROM tax_codes t
    WHERE t.id = (r.tax_config ->> 'tax_code_id')::char(26)
      AND t.namespace = r.namespace
  );

DO $$
DECLARE
  unresolved_count int;
BEGIN
  SELECT count(*) INTO unresolved_count
  FROM billing_invoice_lines r
  JOIN _standard_invoice_line_tax_scope s ON s.id = r.id AND s.invoice_id = r.invoice_id
  WHERE r.tax_code_id IS NULL
    AND r.tax_config ->> 'tax_code_id' IS NOT NULL;

  IF unresolved_count > 0 THEN
    RAISE EXCEPTION 'standard invoice line tax config backfill: % row(s) contain an embedded tax_code_id that cannot be resolved to a tax code in the same namespace', unresolved_count;
  END IF;
END $$;

DO $$
DECLARE
  invalid_reference_count int;
BEGIN
  SELECT count(*) INTO invalid_reference_count
  FROM billing_invoice_lines r
  JOIN _standard_invoice_line_tax_scope s ON s.id = r.id AND s.invoice_id = r.invoice_id
  WHERE r.tax_code_id IS NOT NULL
    AND NOT EXISTS (
      SELECT 1
      FROM tax_codes t
      WHERE t.id = r.tax_code_id
        AND t.namespace = r.namespace
    );

  IF invalid_reference_count > 0 THEN
    RAISE EXCEPTION 'standard invoice line tax config backfill: % row(s) reference a missing or cross-namespace tax code', invalid_reference_count;
  END IF;
END $$;

CREATE TEMP TABLE _standard_invoice_line_tax_pairs AS
SELECT DISTINCT
    r.namespace,
    btrim(r.tax_config -> 'stripe' ->> 'code') AS stripe_code
FROM billing_invoice_lines r
JOIN _standard_invoice_line_tax_scope s ON s.id = r.id AND s.invoice_id = r.invoice_id
WHERE r.tax_code_id IS NULL
  AND btrim(r.tax_config -> 'stripe' ->> 'code') <> '';

CREATE INDEX ON _standard_invoice_line_tax_pairs (namespace, stripe_code);

DO $$
DECLARE
  invalid_count int;
  invalid_codes text;
BEGIN
  SELECT count(*), string_agg(DISTINCT stripe_code, ', ')
    INTO invalid_count, invalid_codes
  FROM _standard_invoice_line_tax_pairs
  WHERE stripe_code !~ '^txcd_[0-9]{8}$';

  IF invalid_count > 0 THEN
    RAISE EXCEPTION 'standard invoice line tax config backfill: % (namespace, code) pair(s) carry a non-Stripe tax code; correct or clear these tax configs and re-run. Offending codes: %', invalid_count, invalid_codes;
  END IF;
END $$;

CREATE TEMP TABLE _standard_invoice_line_tax_map (
    namespace character varying NOT NULL,
    stripe_code text NOT NULL,
    winner_id char(26) NOT NULL,
    created boolean NOT NULL DEFAULT FALSE
);

INSERT INTO _standard_invoice_line_tax_map (namespace, stripe_code, winner_id)
WITH expanded AS (
    SELECT
        t.id,
        t.namespace,
        t.created_at,
        m ->> 'tax_code' AS app_tax_code,
        COALESCE(t.annotations ->> 'managed_by', '') = 'system' AS is_system
    FROM tax_codes t,
         LATERAL jsonb_array_elements(
           CASE
             WHEN jsonb_typeof(t.app_mappings) = 'array' THEN t.app_mappings
             ELSE '[]'::jsonb
           END
         ) AS m
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
    JOIN _standard_invoice_line_tax_pairs p
      ON p.namespace = e.namespace
     AND p.stripe_code = e.app_tax_code
)
SELECT namespace, app_tax_code, id
FROM ranked
WHERE rn = 1;

DO $$
DECLARE
  orphaned_count int;
  orphaned text;
BEGIN
  SELECT count(*), string_agg(DISTINCT t.namespace || ':' || t.key, ', ')
    INTO orphaned_count, orphaned
  FROM _standard_invoice_line_tax_pairs p
  JOIN tax_codes t
    ON t.namespace = p.namespace
   AND t.key = 'stripe_' || p.stripe_code
   AND t.deleted_at IS NULL
  LEFT JOIN _standard_invoice_line_tax_map m
    ON m.namespace = p.namespace
   AND m.stripe_code = p.stripe_code
  WHERE m.winner_id IS NULL;

  IF orphaned_count > 0 THEN
    RAISE EXCEPTION 'standard invoice line tax config backfill: % orphaned auto-key(s); a live tax code entity holds the stripe_ key without the matching app mapping: %', orphaned_count, orphaned;
  END IF;
END $$;

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
    FROM _standard_invoice_line_tax_pairs p
    WHERE NOT EXISTS (
        SELECT 1
        FROM _standard_invoice_line_tax_map m
        WHERE m.namespace = p.namespace
          AND m.stripe_code = p.stripe_code
    )
    RETURNING id, namespace, app_mappings
)
INSERT INTO _standard_invoice_line_tax_map (namespace, stripe_code, winner_id, created)
SELECT i.namespace, i.app_mappings -> 0 ->> 'tax_code', i.id, TRUE
FROM inserted i;

UPDATE billing_invoice_lines r
SET tax_code_id = m.winner_id,
    tax_config = jsonb_set(
        r.tax_config,
        '{tax_code_id}',
        to_jsonb(m.winner_id::text)
    )
FROM _standard_invoice_line_tax_map m,
     _standard_invoice_line_tax_scope s
WHERE s.id = r.id
  AND s.invoice_id = r.invoice_id
  AND r.namespace = m.namespace
  AND r.tax_code_id IS NULL
  AND btrim(r.tax_config -> 'stripe' ->> 'code') = m.stripe_code;

UPDATE billing_invoice_lines r
SET tax_behavior = r.tax_config ->> 'behavior'
FROM _standard_invoice_line_tax_scope s
WHERE s.id = r.id
  AND s.invoice_id = r.invoice_id
  AND r.tax_behavior IS NULL
  AND r.tax_config ->> 'behavior' IS NOT NULL;

UPDATE billing_invoice_lines r
SET tax_config =
  COALESCE(NULLIF(r.tax_config, 'null'::jsonb), '{}'::jsonb)
  || CASE
    WHEN r.tax_code_id IS NOT NULL
      AND r.tax_config ->> 'tax_code_id' IS NULL
    THEN jsonb_build_object('tax_code_id', r.tax_code_id::text)
    ELSE '{}'::jsonb
  END
  || CASE
    WHEN r.tax_behavior IS NOT NULL
      AND r.tax_config ->> 'behavior' IS NULL
    THEN jsonb_build_object('behavior', r.tax_behavior::text)
    ELSE '{}'::jsonb
  END
FROM _standard_invoice_line_tax_scope s
WHERE s.id = r.id
  AND s.invoice_id = r.invoice_id
  AND (
    (
      r.tax_code_id IS NOT NULL
      AND r.tax_config ->> 'tax_code_id' IS NULL
    )
    OR (
      r.tax_behavior IS NOT NULL
      AND r.tax_config ->> 'behavior' IS NULL
    )
  );

DO $$
DECLARE
  tax_code_mismatches int;
  behavior_mismatches int;
  invalid_reference_count int;
BEGIN
  SELECT count(*) INTO tax_code_mismatches
  FROM billing_invoice_lines r
  JOIN _standard_invoice_line_tax_scope s ON s.id = r.id AND s.invoice_id = r.invoice_id
  WHERE r.tax_code_id::text IS DISTINCT FROM r.tax_config ->> 'tax_code_id'
     OR (
       r.tax_code_id IS NULL
       AND NULLIF(btrim(r.tax_config -> 'stripe' ->> 'code'), '') IS NOT NULL
     );

  SELECT count(*) INTO behavior_mismatches
  FROM billing_invoice_lines r
  JOIN _standard_invoice_line_tax_scope s ON s.id = r.id AND s.invoice_id = r.invoice_id
  WHERE r.tax_behavior IS DISTINCT FROM r.tax_config ->> 'behavior';

  SELECT count(*) INTO invalid_reference_count
  FROM billing_invoice_lines r
  JOIN _standard_invoice_line_tax_scope s ON s.id = r.id AND s.invoice_id = r.invoice_id
  WHERE r.tax_code_id IS NOT NULL
    AND NOT EXISTS (
      SELECT 1
      FROM tax_codes t
      WHERE t.id = r.tax_code_id
        AND t.namespace = r.namespace
    );

  IF tax_code_mismatches > 0
    OR behavior_mismatches > 0
    OR invalid_reference_count > 0
  THEN
    RAISE EXCEPTION 'standard invoice line tax config backfill: billing_invoice_lines still has % standard row(s) with inconsistent tax code representations, % standard row(s) with inconsistent tax behavior representations, and % standard row(s) with invalid normalized tax code references', tax_code_mismatches, behavior_mismatches, invalid_reference_count;
  END IF;
END $$;

DO $$
DECLARE
  scoped_count int;
  attached int;
  created_count int;
BEGIN
  SELECT count(*) INTO scoped_count FROM _standard_invoice_line_tax_scope;
  SELECT count(*) INTO attached FROM _standard_invoice_line_tax_map WHERE NOT created;
  SELECT count(*) INTO created_count FROM _standard_invoice_line_tax_map WHERE created;
  RAISE NOTICE 'standard invoice line tax config backfill: checked % row(s), resolved % (namespace, code) pair(s); % attached to existing tax codes, % created new tax codes', scoped_count, attached + created_count, attached, created_count;
END $$;

DROP TABLE _standard_invoice_line_tax_scope;
DROP TABLE _standard_invoice_line_tax_pairs;
DROP TABLE _standard_invoice_line_tax_map;

COMMIT;
