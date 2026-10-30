-- create index "subscription_namespace_updated_at" to table: "subscriptions"
CREATE INDEX "subscription_namespace_updated_at" ON "subscriptions" ("namespace", "updated_at");
