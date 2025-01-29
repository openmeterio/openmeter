-- let's delete all subscription items where the usagePeriod is less than 1 hour
UPDATE
    subscription_items
SET
    deleted_at = NOW()
WHERE
    (entitlement_template ->> 'usagePeriod') :: INTERVAL < 'PT1H' :: INTERVAL;

-- let's delete all plan rate cards where the usagePeriod is less than 1 hour
UPDATE
    plan_rate_cards
SET
    deleted_at = NOW()
WHERE
    (entitlement_template ->> 'usagePeriod') :: INTERVAL < 'PT1H' :: INTERVAL;