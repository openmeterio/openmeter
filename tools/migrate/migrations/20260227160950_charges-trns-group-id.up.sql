-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD COLUMN "authorized_transaction_group_id" character(26) NULL, ADD COLUMN "settled_transaction_group_id" character(26) NULL;
