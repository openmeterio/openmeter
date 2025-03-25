-- reverse: modify "entitlements" table
ALTER TABLE "entitlements" ADD COLUMN "subscription_managed" boolean NULL;
