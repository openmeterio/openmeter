-- create "addons" table
CREATE TABLE "addons" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "key" character varying NOT NULL,
  "version" bigint NOT NULL,
  "currency" character varying NOT NULL DEFAULT 'USD',
  "effective_from" timestamptz NULL,
  "effective_to" timestamptz NULL,
  "annotations" jsonb NULL,
  PRIMARY KEY ("id")
);
-- create index "addon_annotations" to table: "addons"
CREATE INDEX "addon_annotations" ON "addons" USING gin ("annotations");
-- create index "addon_id" to table: "addons"
CREATE UNIQUE INDEX "addon_id" ON "addons" ("id");
-- create index "addon_namespace" to table: "addons"
CREATE INDEX "addon_namespace" ON "addons" ("namespace");
-- create index "addon_namespace_id" to table: "addons"
CREATE UNIQUE INDEX "addon_namespace_id" ON "addons" ("namespace", "id");
-- create index "addon_namespace_key_deleted_at" to table: "addons"
CREATE UNIQUE INDEX "addon_namespace_key_deleted_at" ON "addons" ("namespace", "key", "deleted_at");
-- create index "addon_namespace_key_version" to table: "addons"
CREATE UNIQUE INDEX "addon_namespace_key_version" ON "addons" ("namespace", "key", "version") WHERE (deleted_at IS NULL);
-- create "addon_rate_cards" table
CREATE TABLE "addon_rate_cards" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "key" character varying NOT NULL,
  "type" character varying NOT NULL,
  "feature_key" character varying NULL,
  "entitlement_template" jsonb NULL,
  "tax_config" jsonb NULL,
  "billing_cadence" character varying NULL,
  "price" jsonb NULL,
  "discounts" jsonb NULL,
  "addon_id" character(26) NOT NULL,
  "feature_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "addon_rate_cards_addons_ratecards" FOREIGN KEY ("addon_id") REFERENCES "addons" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "addon_rate_cards_features_addon_ratecard" FOREIGN KEY ("feature_id") REFERENCES "features" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "addonratecard_addon_id_feature_key" to table: "addon_rate_cards"
CREATE UNIQUE INDEX "addonratecard_addon_id_feature_key" ON "addon_rate_cards" ("addon_id", "feature_key") WHERE (deleted_at IS NULL);
-- create index "addonratecard_addon_id_key" to table: "addon_rate_cards"
CREATE UNIQUE INDEX "addonratecard_addon_id_key" ON "addon_rate_cards" ("addon_id", "key") WHERE (deleted_at IS NULL);
-- create index "addonratecard_id" to table: "addon_rate_cards"
CREATE UNIQUE INDEX "addonratecard_id" ON "addon_rate_cards" ("id");
-- create index "addonratecard_namespace" to table: "addon_rate_cards"
CREATE INDEX "addonratecard_namespace" ON "addon_rate_cards" ("namespace");
-- create index "addonratecard_namespace_id" to table: "addon_rate_cards"
CREATE UNIQUE INDEX "addonratecard_namespace_id" ON "addon_rate_cards" ("namespace", "id");
-- create index "addonratecard_namespace_key_deleted_at" to table: "addon_rate_cards"
CREATE UNIQUE INDEX "addonratecard_namespace_key_deleted_at" ON "addon_rate_cards" ("namespace", "key", "deleted_at");
