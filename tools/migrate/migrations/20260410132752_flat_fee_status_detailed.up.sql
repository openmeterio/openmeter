-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD COLUMN "status_detailed" character varying NULL;
UPDATE "charge_flat_fees" SET "status_detailed" = "status";
-- atlas:nolint MF104
ALTER TABLE "charge_flat_fees" ALTER COLUMN "status_detailed" SET NOT NULL;
