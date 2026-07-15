-- modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" ADD COLUMN "currency" character varying NULL;
-- modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" ADD COLUMN "currency" character varying NULL;
