-- modify "entitlements" table
ALTER TABLE "entitlements" ADD COLUMN "subscription_managed" boolean NULL, ADD COLUMN "entitlement_subscription_item" character(26) NULL;
-- create "subscription_items" table
CREATE TABLE "subscription_items" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "metadata" jsonb NULL,
  "active_from" timestamptz NOT NULL,
  "active_to" timestamptz NULL,
  "key" character varying NOT NULL,
  "active_from_override_relative_to_phase_start" character varying NULL,
  "active_to_override_relative_to_phase_start" character varying NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "feature_key" character varying NULL,
  "entitlement_template" jsonb NULL,
  "tax_config" jsonb NULL,
  "billing_cadence" character varying NULL,
  "price" jsonb NULL,
  "entitlement_id" character(26) NULL,
  "phase_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "subscriptionitem_id" to table: "subscription_items"
CREATE UNIQUE INDEX "subscriptionitem_id" ON "subscription_items" ("id");
-- create index "subscriptionitem_namespace" to table: "subscription_items"
CREATE INDEX "subscriptionitem_namespace" ON "subscription_items" ("namespace");
-- create index "subscriptionitem_namespace_id" to table: "subscription_items"
CREATE INDEX "subscriptionitem_namespace_id" ON "subscription_items" ("namespace", "id");
-- create index "subscriptionitem_namespace_phase_id_key" to table: "subscription_items"
CREATE INDEX "subscriptionitem_namespace_phase_id_key" ON "subscription_items" ("namespace", "phase_id", "key");
-- create "subscription_phases" table
CREATE TABLE "subscription_phases" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "metadata" jsonb NULL,
  "key" character varying NOT NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "active_from" timestamptz NOT NULL,
  "subscription_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "subscriptionphase_id" to table: "subscription_phases"
CREATE UNIQUE INDEX "subscriptionphase_id" ON "subscription_phases" ("id");
-- create index "subscriptionphase_namespace" to table: "subscription_phases"
CREATE INDEX "subscriptionphase_namespace" ON "subscription_phases" ("namespace");
-- create index "subscriptionphase_namespace_id" to table: "subscription_phases"
CREATE INDEX "subscriptionphase_namespace_id" ON "subscription_phases" ("namespace", "id");
-- create index "subscriptionphase_namespace_subscription_id" to table: "subscription_phases"
CREATE INDEX "subscriptionphase_namespace_subscription_id" ON "subscription_phases" ("namespace", "subscription_id");
-- create index "subscriptionphase_namespace_subscription_id_key" to table: "subscription_phases"
CREATE INDEX "subscriptionphase_namespace_subscription_id_key" ON "subscription_phases" ("namespace", "subscription_id", "key");
-- create "subscriptions" table
CREATE TABLE "subscriptions" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "metadata" jsonb NULL,
  "active_from" timestamptz NOT NULL,
  "active_to" timestamptz NULL,
  "plan_key" character varying NOT NULL,
  "plan_version" bigint NOT NULL,
  "currency" character varying NOT NULL,
  "customer_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "subscription_id" to table: "subscriptions"
CREATE UNIQUE INDEX "subscription_id" ON "subscriptions" ("id");
-- create index "subscription_namespace" to table: "subscriptions"
CREATE INDEX "subscription_namespace" ON "subscriptions" ("namespace");
-- create index "subscription_namespace_customer_id" to table: "subscriptions"
CREATE INDEX "subscription_namespace_customer_id" ON "subscriptions" ("namespace", "customer_id");
-- create index "subscription_namespace_id" to table: "subscriptions"
CREATE INDEX "subscription_namespace_id" ON "subscriptions" ("namespace", "id");
-- modify "entitlements" table
ALTER TABLE "entitlements" ADD
 CONSTRAINT "entitlements_subscription_items_subscription_item" FOREIGN KEY ("entitlement_subscription_item") REFERENCES "subscription_items" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "subscription_items" table
ALTER TABLE "subscription_items" ADD
 CONSTRAINT "subscription_items_entitlements_entitlement" FOREIGN KEY ("entitlement_id") REFERENCES "entitlements" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD
 CONSTRAINT "subscription_items_subscription_phases_items" FOREIGN KEY ("phase_id") REFERENCES "subscription_phases" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- modify "subscription_phases" table
ALTER TABLE "subscription_phases" ADD
 CONSTRAINT "subscription_phases_subscriptions_phases" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- modify "subscriptions" table
ALTER TABLE "subscriptions" ADD
 CONSTRAINT "subscriptions_customers_subscription" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
