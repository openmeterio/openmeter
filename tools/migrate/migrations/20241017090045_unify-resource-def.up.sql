-- modify "apps" table
ALTER TABLE "apps" ALTER COLUMN "description" DROP NOT NULL;
-- modify "billing_profiles" table
-- atlas:nolint MF103
ALTER TABLE "billing_profiles" ADD COLUMN "name" character varying NOT NULL, ADD COLUMN "description" character varying NULL;
-- modify "customers" table
ALTER TABLE "customers" ADD COLUMN "description" character varying NULL;
