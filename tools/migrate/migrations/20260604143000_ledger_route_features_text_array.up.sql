-- modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" DROP COLUMN "features", ADD COLUMN "features" text[] NULL;
