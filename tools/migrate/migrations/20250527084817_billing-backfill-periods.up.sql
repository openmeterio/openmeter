UPDATE billing_invoices
SET period_start = (
    SELECT
        MIN(period_start)
    FROM billing_invoice_lines
    WHERE
        invoice_id = billing_invoices.id AND
        deleted_at IS NULL AND
        status = 'valid'
)
WHERE period_start IS NULL AND status = 'gathering' AND deleted_at is NULL;

UPDATE billing_invoices
SET period_end = (
    SELECT
        MAX(period_end)
    FROM billing_invoice_lines
    WHERE
        invoice_id = billing_invoices.id AND
        deleted_at IS NULL AND
        status = 'valid'
)
WHERE period_end IS NULL AND status = 'gathering' AND deleted_at is NULL;
