-- reverse: modify "ledger_sub_account_routes" table
-- Fail loudly if V2 routing rows exist. Dropping tax_behavior while V2 routing keys
-- remain would leave rows whose routing_key references a dimension no longer stored
-- in the table.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM ledger_sub_account_routes
        WHERE routing_key_version = 'v2'
    ) THEN
        RAISE EXCEPTION 'cannot rollback: V2 routing key rows exist; downgrade routes first';
    END IF;
END $$;

ALTER TABLE "ledger_sub_account_routes" DROP COLUMN "tax_behavior";
