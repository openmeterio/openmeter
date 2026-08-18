-- modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" ADD CONSTRAINT "addon_rate_card_feature_reference" CHECK (((feature_key IS NULL) AND (feature_id IS NULL)) OR ((feature_key IS NOT NULL) AND ((feature_key)::text <> ''::text) AND (feature_id IS NOT NULL) AND (feature_id <> ''::bpchar)));
-- modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" ADD CONSTRAINT "plan_rate_card_feature_reference" CHECK (((feature_key IS NULL) AND (feature_id IS NULL)) OR ((feature_key IS NOT NULL) AND ((feature_key)::text <> ''::text) AND (feature_id IS NOT NULL) AND (feature_id <> ''::bpchar)));
