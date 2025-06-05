-- modify "balance_snapshots" table
ALTER TABLE "balance_snapshots" DROP CONSTRAINT "balance_snapshots_entitlements_balance_snapshot", ADD CONSTRAINT "balance_snapshots_entitlements_balance_snapshot" FOREIGN KEY ("owner_id") REFERENCES "entitlements" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- modify "grants" table
ALTER TABLE "grants" DROP CONSTRAINT "grants_entitlements_grant", ADD CONSTRAINT "grants_entitlements_grant" FOREIGN KEY ("owner_id") REFERENCES "entitlements" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- modify "subscription_phases" table
ALTER TABLE "subscription_phases" DROP CONSTRAINT "subscription_phases_subscriptions_phases", ADD CONSTRAINT "subscription_phases_subscriptions_phases" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_items_subscription_phases_items", ADD CONSTRAINT "subscription_items_subscription_phases_items" FOREIGN KEY ("phase_id") REFERENCES "subscription_phases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- modify "usage_resets" table
ALTER TABLE "usage_resets" DROP CONSTRAINT "usage_resets_entitlements_usage_reset", ADD CONSTRAINT "usage_resets_entitlements_usage_reset" FOREIGN KEY ("entitlement_id") REFERENCES "entitlements" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;

-- create trigger to delete entitlement when subscription_item is deleted
CREATE OR REPLACE FUNCTION delete_entitlement_on_subscription_item_delete()
RETURNS TRIGGER AS $$
BEGIN
    -- Delete the entitlement if it exists and is referenced by the deleted subscription_item
    IF OLD.entitlement_id IS NOT NULL THEN
        DELETE FROM entitlements WHERE id = OLD.entitlement_id;
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_delete_entitlement_on_subscription_item_delete
    AFTER DELETE ON subscription_items
    FOR EACH ROW
    EXECUTE FUNCTION delete_entitlement_on_subscription_item_delete();
