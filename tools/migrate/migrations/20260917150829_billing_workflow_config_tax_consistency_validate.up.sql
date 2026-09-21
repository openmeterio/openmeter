-- Validate the billing-workflow-config tax consistency checks separately from the
-- data repair so PostgreSQL does not hold validation locks for its duration.
ALTER TABLE "billing_workflow_configs"
  VALIDATE CONSTRAINT "billing_workflow_config_tax_behavior_consistency";

ALTER TABLE "billing_workflow_configs"
  VALIDATE CONSTRAINT "billing_workflow_config_tax_code_consistency";

-- CHECK constraints cannot inspect tax_codes, so verify the cross-table
-- reference and provider-identity invariants explicitly.
DO $$
DECLARE
  invalid_reference_count int;
  mismatched_identity_count int;
BEGIN
  SELECT count(*) INTO invalid_reference_count
  FROM billing_workflow_configs r
  WHERE r.tax_code_id IS NOT NULL
    AND NOT EXISTS (
      SELECT 1
      FROM tax_codes t
      WHERE t.id = r.tax_code_id
        AND t.namespace = r.namespace
        AND t.deleted_at IS NULL
    );

  SELECT count(*) INTO mismatched_identity_count
  FROM billing_workflow_configs r
  WHERE r.tax_code_id IS NOT NULL
    AND NULLIF(btrim(r.invoice_default_tax_settings -> 'stripe' ->> 'code'), '') IS NOT NULL
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
        AND m ->> 'tax_code' = r.invoice_default_tax_settings -> 'stripe' ->> 'code'
    );

  IF invalid_reference_count > 0 OR mismatched_identity_count > 0 THEN
    RAISE EXCEPTION 'billing workflow config tax validation: billing_workflow_configs has % row(s) with invalid normalized tax code references and % row(s) with mismatched tax identities', invalid_reference_count, mismatched_identity_count;
  END IF;
END $$;
