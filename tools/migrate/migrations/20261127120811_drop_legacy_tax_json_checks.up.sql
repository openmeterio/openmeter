-- modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" DROP CONSTRAINT "addon_rate_card_tax_behavior_consistency", DROP CONSTRAINT "addon_rate_card_tax_code_consistency";
-- modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" DROP CONSTRAINT "billing_workflow_config_tax_behavior_consistency", DROP CONSTRAINT "billing_workflow_config_tax_code_consistency";
-- modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" DROP CONSTRAINT "plan_rate_card_tax_behavior_consistency", DROP CONSTRAINT "plan_rate_card_tax_code_consistency";
-- modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_item_tax_behavior_consistency", DROP CONSTRAINT "subscription_item_tax_code_consistency";
