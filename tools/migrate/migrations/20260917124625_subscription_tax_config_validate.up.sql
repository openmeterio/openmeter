-- Validate the subscription-item tax consistency checks separately from the
-- data backfill so PostgreSQL does not hold validation locks for its duration.
ALTER TABLE "subscription_items"
  VALIDATE CONSTRAINT "subscription_item_tax_behavior_consistency";

ALTER TABLE "subscription_items"
  VALIDATE CONSTRAINT "subscription_item_tax_code_consistency";
