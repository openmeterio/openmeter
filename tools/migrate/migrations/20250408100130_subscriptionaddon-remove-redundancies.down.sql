-- reverse: modify "subscription_addon_rate_cards" table
ALTER TABLE "subscription_addon_rate_cards" ADD COLUMN "key" character varying NOT NULL, ADD COLUMN "description" character varying NULL, ADD COLUMN "name" character varying NOT NULL;
