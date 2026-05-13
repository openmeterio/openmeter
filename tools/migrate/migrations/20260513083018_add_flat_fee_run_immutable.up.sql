-- modify "charge_flat_fee_runs" table
ALTER TABLE "charge_flat_fee_runs" ADD COLUMN "immutable" boolean NULL;

UPDATE "charge_flat_fee_runs"
SET "immutable" = false
WHERE "immutable" IS NULL;

ALTER TABLE "charge_flat_fee_runs" ALTER COLUMN "immutable" SET NOT NULL;
