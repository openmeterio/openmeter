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
  PRIMARY KEY ("id"),
  CONSTRAINT "subscriptions_customers_subscription" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "subscription_id" to table: "subscriptions"
CREATE UNIQUE INDEX "subscription_id" ON "subscriptions" ("id");
-- create index "subscription_namespace" to table: "subscriptions"
CREATE INDEX "subscription_namespace" ON "subscriptions" ("namespace");
-- create index "subscription_namespace_customer_id" to table: "subscriptions"
CREATE INDEX "subscription_namespace_customer_id" ON "subscriptions" ("namespace", "customer_id");
-- create index "subscription_namespace_id" to table: "subscriptions"
CREATE INDEX "subscription_namespace_id" ON "subscriptions" ("namespace", "id");
-- create "prices" table
CREATE TABLE "prices" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "active_from" timestamptz NOT NULL,
  "active_to" timestamptz NULL,
  "key" character varying NOT NULL,
  "phase_key" character varying NOT NULL,
  "item_key" character varying NOT NULL,
  "value" character varying NOT NULL,
  "subscription_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "prices_subscriptions_prices" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "price_id" to table: "prices"
CREATE UNIQUE INDEX "price_id" ON "prices" ("id");
-- create index "price_namespace" to table: "prices"
CREATE INDEX "price_namespace" ON "prices" ("namespace");
-- create index "price_namespace_id" to table: "prices"
CREATE INDEX "price_namespace_id" ON "prices" ("namespace", "id");
-- create index "price_namespace_subscription_id" to table: "prices"
CREATE INDEX "price_namespace_subscription_id" ON "prices" ("namespace", "subscription_id");
-- create index "price_namespace_subscription_id_key" to table: "prices"
CREATE INDEX "price_namespace_subscription_id_key" ON "prices" ("namespace", "subscription_id", "key");
-- create "subscription_entitlements" table
CREATE TABLE "subscription_entitlements" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "subscription_phase_key" character varying NOT NULL,
  "subscription_item_key" character varying NOT NULL,
  "entitlement_id" character(26) NOT NULL,
  "subscription_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_entitlements_entitlements_subscription" FOREIGN KEY ("entitlement_id") REFERENCES "entitlements" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "subscription_entitlements_subscriptions_entitlements" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "subscription_entitlements_entitlement_id_key" to table: "subscription_entitlements"
CREATE UNIQUE INDEX "subscription_entitlements_entitlement_id_key" ON "subscription_entitlements" ("entitlement_id");
-- create index "subscriptionentitlement_id" to table: "subscription_entitlements"
CREATE UNIQUE INDEX "subscriptionentitlement_id" ON "subscription_entitlements" ("id");
-- create index "subscriptionentitlement_namespace" to table: "subscription_entitlements"
CREATE INDEX "subscriptionentitlement_namespace" ON "subscription_entitlements" ("namespace");
-- create index "subscriptionentitlement_namespace_entitlement_id" to table: "subscription_entitlements"
CREATE INDEX "subscriptionentitlement_namespace_entitlement_id" ON "subscription_entitlements" ("namespace", "entitlement_id");
-- create index "subscriptionentitlement_namespace_id" to table: "subscription_entitlements"
CREATE INDEX "subscriptionentitlement_namespace_id" ON "subscription_entitlements" ("namespace", "id");
-- create index "subscriptionentitlement_namespace_subscription_id" to table: "subscription_entitlements"
CREATE INDEX "subscriptionentitlement_namespace_subscription_id" ON "subscription_entitlements" ("namespace", "subscription_id");
-- create index "subscriptionentitlement_namespace_subscription_id_subscription_" to table: "subscription_entitlements"
CREATE INDEX "subscriptionentitlement_namespace_subscription_id_subscription_" ON "subscription_entitlements" ("namespace", "subscription_id", "subscription_phase_key", "subscription_item_key");
-- create "subscription_patches" table
CREATE TABLE "subscription_patches" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "metadata" jsonb NULL,
  "applied_at" timestamptz NOT NULL,
  "batch_index" bigint NOT NULL,
  "operation" character varying NOT NULL,
  "path" character varying NOT NULL,
  "subscription_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_patches_subscriptions_subscription_patches" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "subscriptionpatch_id" to table: "subscription_patches"
CREATE UNIQUE INDEX "subscriptionpatch_id" ON "subscription_patches" ("id");
-- create index "subscriptionpatch_namespace" to table: "subscription_patches"
CREATE INDEX "subscriptionpatch_namespace" ON "subscription_patches" ("namespace");
-- create index "subscriptionpatch_namespace_id" to table: "subscription_patches"
CREATE INDEX "subscriptionpatch_namespace_id" ON "subscription_patches" ("namespace", "id");
-- create index "subscriptionpatch_namespace_subscription_id" to table: "subscription_patches"
CREATE INDEX "subscriptionpatch_namespace_subscription_id" ON "subscription_patches" ("namespace", "subscription_id");
-- create "subscription_patch_value_add_items" table
CREATE TABLE "subscription_patch_value_add_items" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "phase_key" character varying NOT NULL,
  "item_key" character varying NOT NULL,
  "feature_key" character varying NULL,
  "create_entitlement_entitlement_type" character varying NULL,
  "create_entitlement_issue_after_reset" double precision NULL,
  "create_entitlement_issue_after_reset_priority" smallint NULL,
  "create_entitlement_is_soft_limit" boolean NULL,
  "create_entitlement_preserve_overage_at_reset" boolean NULL,
  "create_entitlement_usage_period_iso_duration" character varying NULL,
  "create_entitlement_config" jsonb NULL,
  "create_price_key" character varying NULL,
  "create_price_value" numeric NULL,
  "subscription_patch_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_patch_value_add_items_subscription_patches_value_a" FOREIGN KEY ("subscription_patch_id") REFERENCES "subscription_patches" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "subscription_patch_value_add_items_subscription_patch_id_key" to table: "subscription_patch_value_add_items"
CREATE UNIQUE INDEX "subscription_patch_value_add_items_subscription_patch_id_key" ON "subscription_patch_value_add_items" ("subscription_patch_id");
-- create index "subscriptionpatchvalueadditem_id" to table: "subscription_patch_value_add_items"
CREATE UNIQUE INDEX "subscriptionpatchvalueadditem_id" ON "subscription_patch_value_add_items" ("id");
-- create index "subscriptionpatchvalueadditem_namespace" to table: "subscription_patch_value_add_items"
CREATE INDEX "subscriptionpatchvalueadditem_namespace" ON "subscription_patch_value_add_items" ("namespace");
-- create index "subscriptionpatchvalueadditem_namespace_id" to table: "subscription_patch_value_add_items"
CREATE INDEX "subscriptionpatchvalueadditem_namespace_id" ON "subscription_patch_value_add_items" ("namespace", "id");
-- create index "subscriptionpatchvalueadditem_namespace_subscription_patch_id" to table: "subscription_patch_value_add_items"
CREATE INDEX "subscriptionpatchvalueadditem_namespace_subscription_patch_id" ON "subscription_patch_value_add_items" ("namespace", "subscription_patch_id");
-- create "subscription_patch_value_add_phases" table
CREATE TABLE "subscription_patch_value_add_phases" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "phase_key" character varying NOT NULL,
  "start_after_iso" character varying NOT NULL,
  "duration_iso" character varying NOT NULL,
  "create_discount" boolean NOT NULL,
  "create_discount_applies_to" jsonb NULL,
  "subscription_patch_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_patch_value_add_phases_subscription_patches_value_" FOREIGN KEY ("subscription_patch_id") REFERENCES "subscription_patches" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "subscription_patch_value_add_phases_subscription_patch_id_key" to table: "subscription_patch_value_add_phases"
CREATE UNIQUE INDEX "subscription_patch_value_add_phases_subscription_patch_id_key" ON "subscription_patch_value_add_phases" ("subscription_patch_id");
-- create index "subscriptionpatchvalueaddphase_id" to table: "subscription_patch_value_add_phases"
CREATE UNIQUE INDEX "subscriptionpatchvalueaddphase_id" ON "subscription_patch_value_add_phases" ("id");
-- create index "subscriptionpatchvalueaddphase_namespace" to table: "subscription_patch_value_add_phases"
CREATE INDEX "subscriptionpatchvalueaddphase_namespace" ON "subscription_patch_value_add_phases" ("namespace");
-- create index "subscriptionpatchvalueaddphase_namespace_id" to table: "subscription_patch_value_add_phases"
CREATE INDEX "subscriptionpatchvalueaddphase_namespace_id" ON "subscription_patch_value_add_phases" ("namespace", "id");
-- create index "subscriptionpatchvalueaddphase_namespace_subscription_patch_id" to table: "subscription_patch_value_add_phases"
CREATE INDEX "subscriptionpatchvalueaddphase_namespace_subscription_patch_id" ON "subscription_patch_value_add_phases" ("namespace", "subscription_patch_id");
-- create "subscription_patch_value_extend_phases" table
CREATE TABLE "subscription_patch_value_extend_phases" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "phase_key" character varying NOT NULL,
  "extend_duration_iso" character varying NOT NULL,
  "subscription_patch_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_patch_value_extend_phases_subscription_patches_val" FOREIGN KEY ("subscription_patch_id") REFERENCES "subscription_patches" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "subscription_patch_value_extend_phases_subscription_patch_id_ke" to table: "subscription_patch_value_extend_phases"
CREATE UNIQUE INDEX "subscription_patch_value_extend_phases_subscription_patch_id_ke" ON "subscription_patch_value_extend_phases" ("subscription_patch_id");
-- create index "subscriptionpatchvalueextendphase_id" to table: "subscription_patch_value_extend_phases"
CREATE UNIQUE INDEX "subscriptionpatchvalueextendphase_id" ON "subscription_patch_value_extend_phases" ("id");
-- create index "subscriptionpatchvalueextendphase_namespace" to table: "subscription_patch_value_extend_phases"
CREATE INDEX "subscriptionpatchvalueextendphase_namespace" ON "subscription_patch_value_extend_phases" ("namespace");
-- create index "subscriptionpatchvalueextendphase_namespace_id" to table: "subscription_patch_value_extend_phases"
CREATE INDEX "subscriptionpatchvalueextendphase_namespace_id" ON "subscription_patch_value_extend_phases" ("namespace", "id");
-- create index "subscriptionpatchvalueextendphase_namespace_subscription_patch_" to table: "subscription_patch_value_extend_phases"
CREATE INDEX "subscriptionpatchvalueextendphase_namespace_subscription_patch_" ON "subscription_patch_value_extend_phases" ("namespace", "subscription_patch_id");
-- create "subscription_patch_value_remove_phases" table
CREATE TABLE "subscription_patch_value_remove_phases" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "phase_key" character varying NOT NULL,
  "shift_behavior" bigint NOT NULL,
  "subscription_patch_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_patch_value_remove_phases_subscription_patches_val" FOREIGN KEY ("subscription_patch_id") REFERENCES "subscription_patches" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "subscription_patch_value_remove_phases_subscription_patch_id_ke" to table: "subscription_patch_value_remove_phases"
CREATE UNIQUE INDEX "subscription_patch_value_remove_phases_subscription_patch_id_ke" ON "subscription_patch_value_remove_phases" ("subscription_patch_id");
-- create index "subscriptionpatchvalueremovephase_id" to table: "subscription_patch_value_remove_phases"
CREATE UNIQUE INDEX "subscriptionpatchvalueremovephase_id" ON "subscription_patch_value_remove_phases" ("id");
-- create index "subscriptionpatchvalueremovephase_namespace" to table: "subscription_patch_value_remove_phases"
CREATE INDEX "subscriptionpatchvalueremovephase_namespace" ON "subscription_patch_value_remove_phases" ("namespace");
-- create index "subscriptionpatchvalueremovephase_namespace_id" to table: "subscription_patch_value_remove_phases"
CREATE INDEX "subscriptionpatchvalueremovephase_namespace_id" ON "subscription_patch_value_remove_phases" ("namespace", "id");
-- create index "subscriptionpatchvalueremovephase_namespace_subscription_patch_" to table: "subscription_patch_value_remove_phases"
CREATE INDEX "subscriptionpatchvalueremovephase_namespace_subscription_patch_" ON "subscription_patch_value_remove_phases" ("namespace", "subscription_patch_id");
