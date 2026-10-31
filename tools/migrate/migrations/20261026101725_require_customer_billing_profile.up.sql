-- modify "billing_customer_overrides" table
-- The preceding cleanup migration deletes all rows with a NULL billing_profile_id.
-- atlas:nolint MF104
ALTER TABLE "billing_customer_overrides" DROP CONSTRAINT "billing_customer_overrides_billing_profiles_billing_customer_ov", ALTER COLUMN "billing_profile_id" SET NOT NULL, ADD CONSTRAINT "billing_customer_overrides_billing_profiles_billing_customer_ov" FOREIGN KEY ("billing_profile_id") REFERENCES "billing_profiles" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
