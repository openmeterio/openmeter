-- Validate the plan-rate-card tax consistency checks separately from the data
-- backfill so PostgreSQL does not hold validation locks for its duration.
ALTER TABLE "plan_rate_cards"
  VALIDATE CONSTRAINT "plan_rate_card_tax_behavior_consistency";

ALTER TABLE "plan_rate_cards"
  VALIDATE CONSTRAINT "plan_rate_card_tax_code_consistency";
