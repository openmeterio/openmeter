-- reverse: modify "usage_resets" table
ALTER TABLE "usage_resets" DROP CONSTRAINT "usage_resets_entitlements_usage_reset", ADD CONSTRAINT "usage_resets_entitlements_usage_reset" FOREIGN KEY ("entitlement_id") REFERENCES "entitlements" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- reverse: modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_items_subscription_phases_items", ADD CONSTRAINT "subscription_items_subscription_phases_items" FOREIGN KEY ("phase_id") REFERENCES "subscription_phases" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- reverse: modify "subscription_phases" table
ALTER TABLE "subscription_phases" DROP CONSTRAINT "subscription_phases_subscriptions_phases", ADD CONSTRAINT "subscription_phases_subscriptions_phases" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- reverse: modify "grants" table
ALTER TABLE "grants" DROP CONSTRAINT "grants_entitlements_grant", ADD CONSTRAINT "grants_entitlements_grant" FOREIGN KEY ("owner_id") REFERENCES "entitlements" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- reverse: modify "balance_snapshots" table
ALTER TABLE "balance_snapshots" DROP CONSTRAINT "balance_snapshots_entitlements_balance_snapshot", ADD CONSTRAINT "balance_snapshots_entitlements_balance_snapshot" FOREIGN KEY ("owner_id") REFERENCES "entitlements" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
