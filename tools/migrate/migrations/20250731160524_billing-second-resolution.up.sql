-- We need to annotate the previous lines to indicate to subscription sync to ignore them due to the period calculation change.
-- Billing sync will ensure that the lines are continous for a subscription item, thus we will not have a few seconds of gaps
-- in the invoices.

UPDATE billing_invoice_lines
SET
    annotations = CASE
        WHEN annotations IS NULL OR annotations = 'null'::jsonb THEN '{}'::jsonb
        ELSE annotations
    END || jsonb_build_object('billing.subscription.sync.ignore', true)
WHERE
    -- Line type usage based
    "type" = 'usage_based'
    -- Status valid
    AND "status" = 'valid'
    -- InvoiceID belongs to a not gathering invoice
    AND "invoice_id" NOT IN (
        SELECT "id" FROM billing_invoices
        WHERE "status" = 'gathering'
    );
