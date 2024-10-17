-- reverse: modify "customers" table
ALTER TABLE "customers" DROP COLUMN "description";
-- reverse: modify "billing_profiles" table
ALTER TABLE "billing_profiles" DROP COLUMN "description", DROP COLUMN "name";
-- reverse: modify "apps" table
ALTER TABLE "apps" ALTER COLUMN "description" SET NOT NULL;
