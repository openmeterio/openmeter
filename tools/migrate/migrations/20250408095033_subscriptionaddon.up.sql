-- create "subscription_addons" table
CREATE TABLE "subscription_addons" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "addon_id" character(26) NOT NULL,
  "subscription_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_addons_addons_subscription_addons" FOREIGN KEY ("addon_id") REFERENCES "addons" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "subscription_addons_subscriptions_addons" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "subscriptionaddon_id" to table: "subscription_addons"
CREATE UNIQUE INDEX "subscriptionaddon_id" ON "subscription_addons" ("id");
-- create index "subscriptionaddon_namespace" to table: "subscription_addons"
CREATE INDEX "subscriptionaddon_namespace" ON "subscription_addons" ("namespace");
-- create "subscription_addon_quantities" table
CREATE TABLE "subscription_addon_quantities" (
  "id" character(26) NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "active_from" timestamptz NOT NULL,
  "quantity" bigint NOT NULL DEFAULT 1,
  "subscription_addon_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_addon_quantities_subscription_addons_quantities" FOREIGN KEY ("subscription_addon_id") REFERENCES "subscription_addons" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "subscriptionaddonquantity_id" to table: "subscription_addon_quantities"
CREATE UNIQUE INDEX "subscriptionaddonquantity_id" ON "subscription_addon_quantities" ("id");
-- create index "subscriptionaddonquantity_subscription_addon_id" to table: "subscription_addon_quantities"
CREATE INDEX "subscriptionaddonquantity_subscription_addon_id" ON "subscription_addon_quantities" ("subscription_addon_id");
-- create "subscription_addon_rate_cards" table
CREATE TABLE "subscription_addon_rate_cards" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "key" character varying NOT NULL,
  "addon_ratecard_id" character(26) NOT NULL,
  "subscription_addon_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_addon_rate_cards_addon_rate_cards_subscription_add" FOREIGN KEY ("addon_ratecard_id") REFERENCES "addon_rate_cards" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "subscription_addon_rate_cards_subscription_addons_rate_cards" FOREIGN KEY ("subscription_addon_id") REFERENCES "subscription_addons" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "subscriptionaddonratecard_id" to table: "subscription_addon_rate_cards"
CREATE UNIQUE INDEX "subscriptionaddonratecard_id" ON "subscription_addon_rate_cards" ("id");
-- create index "subscriptionaddonratecard_namespace" to table: "subscription_addon_rate_cards"
CREATE INDEX "subscriptionaddonratecard_namespace" ON "subscription_addon_rate_cards" ("namespace");
-- create index "subscriptionaddonratecard_namespace_id" to table: "subscription_addon_rate_cards"
CREATE UNIQUE INDEX "subscriptionaddonratecard_namespace_id" ON "subscription_addon_rate_cards" ("namespace", "id");
-- create index "subscriptionaddonratecard_namespace_key_deleted_at" to table: "subscription_addon_rate_cards"
CREATE UNIQUE INDEX "subscriptionaddonratecard_namespace_key_deleted_at" ON "subscription_addon_rate_cards" ("namespace", "key", "deleted_at");
-- create index "subscriptionaddonratecard_subscription_addon_id" to table: "subscription_addon_rate_cards"
CREATE INDEX "subscriptionaddonratecard_subscription_addon_id" ON "subscription_addon_rate_cards" ("subscription_addon_id");
-- create "subscription_addon_rate_card_item_links" table
CREATE TABLE "subscription_addon_rate_card_item_links" (
  "id" character(26) NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "subscription_addon_rate_card_id" character(26) NOT NULL,
  "subscription_item_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_addon_rate_card_item_links_subscription_addon_rate" FOREIGN KEY ("subscription_addon_rate_card_id") REFERENCES "subscription_addon_rate_cards" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "subscription_addon_rate_card_item_links_subscription_items_subs" FOREIGN KEY ("subscription_item_id") REFERENCES "subscription_items" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "subscriptionaddonratecarditemlink_id" to table: "subscription_addon_rate_card_item_links"
CREATE UNIQUE INDEX "subscriptionaddonratecarditemlink_id" ON "subscription_addon_rate_card_item_links" ("id");
-- create index "subscriptionaddonratecarditemlink_subscription_addon_rate_card_" to table: "subscription_addon_rate_card_item_links"
CREATE INDEX "subscriptionaddonratecarditemlink_subscription_addon_rate_card_" ON "subscription_addon_rate_card_item_links" ("subscription_addon_rate_card_id");
-- create index "subscriptionaddonratecarditemlink_subscription_item_id" to table: "subscription_addon_rate_card_item_links"
CREATE INDEX "subscriptionaddonratecarditemlink_subscription_item_id" ON "subscription_addon_rate_card_item_links" ("subscription_item_id");
-- create index "subscriptionaddonratecarditemlink_subscription_item_id_subscrip" to table: "subscription_addon_rate_card_item_links"
CREATE UNIQUE INDEX "subscriptionaddonratecarditemlink_subscription_item_id_subscrip" ON "subscription_addon_rate_card_item_links" ("subscription_item_id", "subscription_addon_rate_card_id") WHERE (deleted_at IS NULL);
