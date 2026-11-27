-- Stamp unstamped lines of Stripe invoices with the legacy stripe.code. The reader used to fall back
-- to it for exactly these lines; after this the stamp alone holds the billed code. tax_config is left
-- as is, as pods of the previous release still read it. Lines without a usable code stay NULL.
-- Gathering lines share this table but are never stamped, so gathering invoices are skipped.
UPDATE "billing_invoice_lines" AS line
SET "invoicing_app_tax_code" = NULLIF(btrim(line."tax_config" -> 'stripe' ->> 'code'), '')
FROM "billing_invoices" AS invoice
JOIN "apps" AS invoicing_app
  ON invoicing_app."namespace" = invoice."namespace"
 AND invoicing_app."id" = invoice."invoicing_app_id"
WHERE invoice."id" = line."invoice_id"
  AND invoice."namespace" = line."namespace"
  AND invoice."type" <> 'gathering'
  AND invoicing_app."type" = 'stripe'
  AND line."invoicing_app_tax_code" IS NULL
  AND NULLIF(btrim(line."tax_config" -> 'stripe' ->> 'code'), '') IS NOT NULL;
