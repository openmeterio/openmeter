-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP COLUMN "settled_transaction_group_id", DROP COLUMN "authorized_transaction_group_id";
