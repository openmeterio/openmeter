-- Close the gap between the gathering and standard backfills, then install
-- table-wide tax consistency checks. Tax-code writers are locked first to match
-- the service's resolution-before-persistence order. Invoice-line writers are
-- blocked only for this final scan and constraint installation.

BEGIN;

LOCK TABLE tax_codes IN SHARE ROW EXCLUSIVE MODE;
LOCK TABLE billing_invoice_lines IN SHARE MODE;

DO $$
DECLARE
  tax_code_mismatches int;
  behavior_mismatches int;
  invalid_reference_count int;
BEGIN
  SELECT count(*) INTO tax_code_mismatches
  FROM billing_invoice_lines
  WHERE tax_code_id::text IS DISTINCT FROM tax_config ->> 'tax_code_id'
     OR (
       tax_code_id IS NULL
       AND NULLIF(btrim(tax_config -> 'stripe' ->> 'code'), '') IS NOT NULL
     );

  SELECT count(*) INTO behavior_mismatches
  FROM billing_invoice_lines
  WHERE tax_behavior IS DISTINCT FROM tax_config ->> 'behavior';

  SELECT count(*) INTO invalid_reference_count
  FROM billing_invoice_lines r
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
    RAISE EXCEPTION 'invoice line tax consistency: billing_invoice_lines still has % row(s) with inconsistent tax code representations, % row(s) with inconsistent tax behavior representations, and % row(s) with invalid normalized tax code references', tax_code_mismatches, behavior_mismatches, invalid_reference_count;
  END IF;
END $$;

-- NOT VALID avoids another historical scan while making every insert and update
-- obey the invariant as soon as this transaction commits.
ALTER TABLE billing_invoice_lines
  ADD CONSTRAINT billing_invoice_line_tax_behavior_consistency
    CHECK (tax_behavior IS NOT DISTINCT FROM tax_config ->> 'behavior') NOT VALID,
  ADD CONSTRAINT billing_invoice_line_tax_code_consistency
    CHECK (
      tax_code_id::text IS NOT DISTINCT FROM tax_config ->> 'tax_code_id'
      AND (
        NULLIF(btrim(tax_config -> 'stripe' ->> 'code'), '') IS NULL
        OR tax_code_id IS NOT NULL
      )
    ) NOT VALID;

COMMIT;
