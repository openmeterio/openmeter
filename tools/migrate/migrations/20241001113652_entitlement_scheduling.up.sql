-- modify "entitlements" table
ALTER TABLE "entitlements" ADD COLUMN "active_from" timestamptz NULL, ADD COLUMN "active_to" timestamptz NULL;
