-- reverse: modify "subscriptions" table
ALTER TABLE "subscriptions" DROP CONSTRAINT "subscriptions_plans_subscriptions", DROP COLUMN "plan_id", ADD COLUMN "plan_version" bigint NOT NULL, ADD COLUMN "plan_key" character varying NOT NULL;
