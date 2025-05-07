-- This down migration removes all rate_card references

-- Remove rate_card references from subscription_items
UPDATE subscription_items
SET ratecard_id = NULL
WHERE ratecard_id IS NOT NULL;

-- Remove rate_card references from addon_rate_cards
UPDATE addon_rate_cards
SET ratecard_id = NULL
WHERE ratecard_id IS NOT NULL;

-- Remove rate_card references from plan_rate_cards
UPDATE plan_rate_cards
SET ratecard_id = NULL
WHERE ratecard_id IS NOT NULL;

