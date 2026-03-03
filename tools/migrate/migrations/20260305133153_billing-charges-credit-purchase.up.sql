-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP COLUMN "status", ADD COLUMN "credit_grant_transaction_group_id" character(26) NULL, ADD COLUMN "credit_granted_at" timestamptz NULL;
