-- Empty selections follow the namespace default and need no persisted assignment.
DELETE FROM "billing_customer_overrides" WHERE "billing_profile_id" IS NULL;
