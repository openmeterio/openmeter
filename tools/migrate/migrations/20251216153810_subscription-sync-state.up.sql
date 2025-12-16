-- create "subscription_billing_sync_states" table
CREATE TABLE "subscription_billing_sync_states" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "has_billables" boolean NOT NULL,
  "synced_at" timestamptz NOT NULL,
  "next_sync_after" timestamptz NULL,
  "subscription_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "subscription_billing_sync_states_subscriptions_billing_sync_sta" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "subscription_billing_sync_states_subscription_id_key" to table: "subscription_billing_sync_states"
CREATE UNIQUE INDEX "subscription_billing_sync_states_subscription_id_key" ON "subscription_billing_sync_states" ("subscription_id");
-- create index "subscriptionbillingsyncstate_id" to table: "subscription_billing_sync_states"
CREATE UNIQUE INDEX "subscriptionbillingsyncstate_id" ON "subscription_billing_sync_states" ("id");
-- create index "subscriptionbillingsyncstate_namespace" to table: "subscription_billing_sync_states"
CREATE INDEX "subscriptionbillingsyncstate_namespace" ON "subscription_billing_sync_states" ("namespace");
-- create index "subscriptionbillingsyncstate_namespace_subscription_id" to table: "subscription_billing_sync_states"
CREATE UNIQUE INDEX "subscriptionbillingsyncstate_namespace_subscription_id" ON "subscription_billing_sync_states" ("namespace", "subscription_id");
