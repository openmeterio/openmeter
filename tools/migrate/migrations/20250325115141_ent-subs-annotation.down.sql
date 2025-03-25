-- This down migration removes the subscription.id annotation from entitlements
-- We'll only run this on entitlements that are subscription_managed = true to be safe

UPDATE entitlements e
SET annotations = e.annotations - 'subscription.id'
WHERE e.subscription_managed = TRUE
AND e.annotations->>'subscription.id' IS NOT NULL;
