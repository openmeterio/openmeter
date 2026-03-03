-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP COLUMN "credit_granted_at", DROP COLUMN "credit_grant_transaction_group_id", ADD COLUMN "status" character varying NOT NULL;
