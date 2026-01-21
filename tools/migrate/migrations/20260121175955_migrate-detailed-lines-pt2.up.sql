
-- 1) Update the default invoice write schema level to 2.
INSERT INTO billing_invoice_write_schema_levels (id, schema_level)
VALUES ('write_schema_level', 2)
ON CONFLICT (id) DO UPDATE
SET schema_level = EXCLUDED.schema_level;

BEGIN;

-- 2) Lock all existing customers for UPDATE (billing customer lock rows).
-- Emulates adapter behavior: upsert lock row then select it FOR UPDATE.
SELECT * FROM billing_customer_locks FOR UPDATE;

-- 3) Migrate invoices for all customers (only affects invoices with schema_level = 1).
SELECT om_func_migrate_customer_invoices_to_schema_level_2(c.customer_id)
FROM billing_customer_locks c;

COMMIT;
