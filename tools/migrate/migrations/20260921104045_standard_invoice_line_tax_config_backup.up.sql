-- Preserve the original standard invoice-line tax representations before the
-- backfill. This migration commits independently so the backup remains
-- available even when the subsequent repair aborts during validation.
BEGIN;

CREATE TABLE IF NOT EXISTS om_migration_backup_20260921104045_invoice_line_tax_config (
    backed_up_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    line_id char(26) NOT NULL,
    namespace character varying NOT NULL,
    invoice_id char(26) NOT NULL,
    tax_config jsonb,
    tax_code_id char(26),
    tax_behavior character varying,
    PRIMARY KEY (backed_up_at, line_id)
);

INSERT INTO om_migration_backup_20260921104045_invoice_line_tax_config (
    line_id,
    namespace,
    invoice_id,
    tax_config,
    tax_code_id,
    tax_behavior
)
SELECT
    l.id,
    l.namespace,
    l.invoice_id,
    l.tax_config,
    l.tax_code_id,
    l.tax_behavior
FROM billing_invoice_lines l
JOIN billing_invoices i
  ON i.id = l.invoice_id
 AND i.namespace = l.namespace
WHERE i.status <> 'gathering';

COMMIT;
