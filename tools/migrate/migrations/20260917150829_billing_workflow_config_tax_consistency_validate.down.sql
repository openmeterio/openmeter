-- Restore the preceding migration's NOT VALID state.
ALTER TABLE "billing_workflow_configs"
  DROP CONSTRAINT "billing_workflow_config_tax_code_consistency",
  DROP CONSTRAINT "billing_workflow_config_tax_behavior_consistency",
  ADD CONSTRAINT "billing_workflow_config_tax_behavior_consistency"
    CHECK (tax_behavior IS NOT DISTINCT FROM invoice_default_tax_settings ->> 'behavior') NOT VALID,
  ADD CONSTRAINT "billing_workflow_config_tax_code_consistency"
    CHECK (
      tax_code_id::text IS NOT DISTINCT FROM invoice_default_tax_settings ->> 'tax_code_id'
      AND (
        NULLIF(btrim(invoice_default_tax_settings -> 'stripe' ->> 'code'), '') IS NULL
        OR tax_code_id IS NOT NULL
      )
    ) NOT VALID;
