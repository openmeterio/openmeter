-- create "rate_cards" table
CREATE TABLE "rate_cards" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "key" character varying NOT NULL,
  "entitlement_template_entitlement_type" character varying NOT NULL,
  "entitlement_template_metadata" jsonb NULL,
  "entitlement_template_is_soft_limit" boolean NULL,
  "entitlement_template_issue_after_reset" double precision NULL,
  "entitlement_template_issue_after_reset_priority" smallint NULL,
  "entitlement_template_preserve_overage_at_reset" boolean NULL,
  "entitlement_template_config" jsonb NULL,
  "entitlement_template_usage_period" character varying NULL,
  "type" character varying NOT NULL,
  "feature_key" character varying NULL,
  "feature_id" character varying NULL,
  "tax_config" jsonb NULL,
  "billing_cadence" character varying NULL,
  "price" jsonb NULL,
  "discounts" jsonb NULL,
  "feature_ratecards" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "rate_cards_features_ratecards" FOREIGN KEY ("feature_ratecards") REFERENCES "features" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "ratecard_id" to table: "rate_cards"
CREATE UNIQUE INDEX "ratecard_id" ON "rate_cards" ("id");
-- create index "ratecard_namespace" to table: "rate_cards"
CREATE INDEX "ratecard_namespace" ON "rate_cards" ("namespace");
-- create index "ratecard_namespace_id" to table: "rate_cards"
CREATE UNIQUE INDEX "ratecard_namespace_id" ON "rate_cards" ("namespace", "id");
-- create index "ratecard_namespace_key_deleted_at" to table: "rate_cards"
CREATE UNIQUE INDEX "ratecard_namespace_key_deleted_at" ON "rate_cards" ("namespace", "key", "deleted_at");
-- modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" ADD COLUMN "ratecard_id" character(26) NULL, ADD
 CONSTRAINT "addon_rate_cards_rate_cards_addon_ratecard" FOREIGN KEY ("ratecard_id") REFERENCES "rate_cards" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "addon_rate_cards_ratecard_id_key" to table: "addon_rate_cards"
CREATE UNIQUE INDEX "addon_rate_cards_ratecard_id_key" ON "addon_rate_cards" ("ratecard_id");
-- modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" ADD COLUMN "ratecard_id" character(26) NULL, ADD
 CONSTRAINT "plan_rate_cards_rate_cards_plan_ratecard" FOREIGN KEY ("ratecard_id") REFERENCES "rate_cards" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "plan_rate_cards_ratecard_id_key" to table: "plan_rate_cards"
CREATE UNIQUE INDEX "plan_rate_cards_ratecard_id_key" ON "plan_rate_cards" ("ratecard_id");
-- modify "subscription_items" table
ALTER TABLE "subscription_items" ADD COLUMN "ratecard_id" character(26) NULL, ADD
 CONSTRAINT "subscription_items_rate_cards_subscription_item" FOREIGN KEY ("ratecard_id") REFERENCES "rate_cards" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "subscription_items_ratecard_id_key" to table: "subscription_items"
CREATE UNIQUE INDEX "subscription_items_ratecard_id_key" ON "subscription_items" ("ratecard_id");
