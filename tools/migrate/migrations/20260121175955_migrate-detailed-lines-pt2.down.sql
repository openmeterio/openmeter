BEGIN;

-- Best-effort rollback: revert the default invoice write schema level back to 1.
-- Data copied during migration is intentionally NOT deleted.
UPDATE billing_invoice_write_schema_levels
SET schema_level = 1
WHERE id = 'write_schema_level';

COMMIT;
