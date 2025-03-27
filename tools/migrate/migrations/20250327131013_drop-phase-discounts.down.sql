-- reverse: modify "plan_phases" table
-- nolint:DS103
ALTER TABLE "plan_phases" ADD COLUMN "discounts" jsonb NULL;
