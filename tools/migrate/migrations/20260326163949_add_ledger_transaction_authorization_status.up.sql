-- modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" ADD COLUMN "transaction_authorization_status" character varying NULL;
