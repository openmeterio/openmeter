-- Frozen reconciliation for Ent baseline 20260709134422.
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

DROP TRIGGER IF EXISTS trigger_delete_entitlement_on_subscription_item_delete ON subscription_items;

CREATE TRIGGER trigger_delete_entitlement_on_subscription_item_delete
    AFTER DELETE ON subscription_items
    FOR EACH ROW
    EXECUTE FUNCTION delete_entitlement_on_subscription_item_delete();
