-- reverse: drop "subscription_addon_rate_cards" table
CREATE TABLE "subscription_addon_rate_cards" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "metadata" jsonb NULL,
  "addon_ratecard_id" character(26) NOT NULL,
  "subscription_addon_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_addon_rate_cards_addon_rate_cards_subscription_add" FOREIGN KEY ("addon_ratecard_id") REFERENCES "addon_rate_cards" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "subscription_addon_rate_cards_subscription_addons_rate_cards" FOREIGN KEY ("subscription_addon_id") REFERENCES "subscription_addons" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
CREATE UNIQUE INDEX "subscriptionaddonratecard_id" ON "subscription_addon_rate_cards" ("id");
CREATE INDEX "subscriptionaddonratecard_namespace" ON "subscription_addon_rate_cards" ("namespace");
CREATE UNIQUE INDEX "subscriptionaddonratecard_namespace_id" ON "subscription_addon_rate_cards" ("namespace", "id");
CREATE INDEX "subscriptionaddonratecard_subscription_addon_id" ON "subscription_addon_rate_cards" ("subscription_addon_id");
-- reverse: drop "subscription_addon_rate_card_item_links" table
CREATE TABLE "subscription_addon_rate_card_item_links" (
  "id" character(26) NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "subscription_addon_rate_card_id" character(26) NOT NULL,
  "subscription_item_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_addon_rate_card_item_links_subscription_addon_rate" FOREIGN KEY ("subscription_addon_rate_card_id") REFERENCES "subscription_addon_rate_cards" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "subscription_addon_rate_card_item_links_subscription_items_subs" FOREIGN KEY ("subscription_item_id") REFERENCES "subscription_items" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
CREATE UNIQUE INDEX "subscriptionaddonratecarditemlink_id" ON "subscription_addon_rate_card_item_links" ("id");
CREATE INDEX "subscriptionaddonratecarditemlink_subscription_addon_rate_card_" ON "subscription_addon_rate_card_item_links" ("subscription_addon_rate_card_id");
CREATE INDEX "subscriptionaddonratecarditemlink_subscription_item_id" ON "subscription_addon_rate_card_item_links" ("subscription_item_id");
CREATE UNIQUE INDEX "subscriptionaddonratecarditemlink_subscription_item_id_subscrip" ON "subscription_addon_rate_card_item_links" ("subscription_item_id", "subscription_addon_rate_card_id") WHERE (deleted_at IS NULL);
