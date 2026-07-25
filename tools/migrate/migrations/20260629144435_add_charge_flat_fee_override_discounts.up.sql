-- modify "charge_flat_fee_overrides" table
ALTER TABLE "charge_flat_fee_overrides" ADD COLUMN "discounts" jsonb NULL;
