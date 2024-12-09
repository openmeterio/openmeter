-- modify "subscriptions" table
ALTER TABLE "subscriptions" DROP COLUMN "plan_key", DROP COLUMN "plan_version", ADD COLUMN "plan_id" character(26) NULL, ADD
 CONSTRAINT "subscriptions_plans_subscriptions" FOREIGN KEY ("plan_id") REFERENCES "plans" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
