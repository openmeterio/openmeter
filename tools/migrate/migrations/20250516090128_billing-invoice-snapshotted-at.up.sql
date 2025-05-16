-- modify "billing_invoices" table
ALTER TABLE "billing_invoices" ADD COLUMN "quantity_snapshoted_at" timestamptz NULL;

-- backfill quantity_snapshoted_at
-- Gathering invoices should not have this value set. In the current workflow this is done in parallel
-- with the darft invoice creation.
UPDATE "billing_invoices"
    SET "quantity_snapshoted_at" = "created_at"
    WHERE "quantity_snapshoted_at" IS NULL AND "status" <> 'gathering';
