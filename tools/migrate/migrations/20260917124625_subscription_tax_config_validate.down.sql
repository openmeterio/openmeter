-- Restore the preceding migration's NOT VALID state.
ALTER TABLE "subscription_items"
  DROP CONSTRAINT "subscription_item_tax_code_consistency",
  DROP CONSTRAINT "subscription_item_tax_behavior_consistency",
  ADD CONSTRAINT "subscription_item_tax_behavior_consistency"
    CHECK (tax_behavior IS NOT DISTINCT FROM tax_config ->> 'behavior') NOT VALID,
  ADD CONSTRAINT "subscription_item_tax_code_consistency"
    CHECK (
      tax_code_id::text IS NOT DISTINCT FROM tax_config ->> 'tax_code_id'
      AND (
        NULLIF(btrim(tax_config -> 'stripe' ->> 'code'), '') IS NULL
        OR tax_code_id IS NOT NULL
      )
    ) NOT VALID;
