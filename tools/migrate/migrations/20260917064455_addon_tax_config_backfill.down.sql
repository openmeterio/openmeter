-- The data backfill is intentionally retained on downgrade. Remove only the
-- forward-write consistency checks.
ALTER TABLE "addon_rate_cards"
  DROP CONSTRAINT "addon_rate_card_tax_code_consistency",
  DROP CONSTRAINT "addon_rate_card_tax_behavior_consistency";
