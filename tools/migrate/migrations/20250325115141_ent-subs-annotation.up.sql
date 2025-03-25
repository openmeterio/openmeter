-- Let's add a migration that select all entitlements that are subscription managed, then
-- 1. find the subscription that it references (through item and phase)
-- 2. add the subscription.id annotation with the subscription's id (if not present) to the entitlement
UPDATE entitlements e
SET annotations = jsonb_set(
    CASE
        WHEN e.annotations IS NULL THEN '{}'::jsonb
        ELSE e.annotations
    END,
    '{subscription.id}',
    to_jsonb(sp.subscription_id)
)
FROM subscription_items si
JOIN subscription_phases sp ON si.phase_id = sp.id
WHERE e.subscription_managed = TRUE
AND e.id = si.entitlement_id
AND (e.annotations IS NULL OR e.annotations->>'subscription.id' IS NULL);
