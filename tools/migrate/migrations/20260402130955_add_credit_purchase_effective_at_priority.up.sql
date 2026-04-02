-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ADD COLUMN "effective_at" timestamptz NULL, ADD COLUMN "priority" bigint NULL;
