-- create "subscription_sync_states_v2" table
CREATE TABLE "subscription_sync_states_v2" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "last_success_state" character varying NOT NULL,
  "last_success_at" timestamptz NOT NULL,
  "subscription_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_sync_state_v2_subscription_fk" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "subscription_sync_states_v2_subscription_id_key" to table: "subscription_sync_states_v2"
CREATE UNIQUE INDEX "subscription_sync_states_v2_subscription_id_key" ON "subscription_sync_states_v2" ("subscription_id");
-- create index "subscriptionsyncstatev2_id" to table: "subscription_sync_states_v2"
CREATE UNIQUE INDEX "subscriptionsyncstatev2_id" ON "subscription_sync_states_v2" ("id");
-- create index "subscriptionsyncstatev2_namespace" to table: "subscription_sync_states_v2"
CREATE INDEX "subscriptionsyncstatev2_namespace" ON "subscription_sync_states_v2" ("namespace");
-- create index "subscriptionsyncstatev2_namespace_subscription_id" to table: "subscription_sync_states_v2"
CREATE UNIQUE INDEX "subscriptionsyncstatev2_namespace_subscription_id" ON "subscription_sync_states_v2" ("namespace", "subscription_id");
