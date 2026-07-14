-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ADD COLUMN "voided_at" timestamptz NULL;
