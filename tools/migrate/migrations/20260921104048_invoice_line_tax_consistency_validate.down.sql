ALTER TABLE billing_invoice_lines
  DROP CONSTRAINT billing_invoice_line_tax_code_consistency,
  DROP CONSTRAINT billing_invoice_line_tax_behavior_consistency,
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
