-- name: GetParentID :one
SELECT l.parent_line_id FROM public.billing_invoice_lines l WHERE l.id = sqlc.arg(line_id)::varchar;

-- name: GetFlatFeeLinesByParentID :many

SELECT l.*, c.per_unit_amount, c.category, c.payment_term, c.index
FROM public.billing_invoice_lines l JOIN public.billing_invoice_flat_fee_line_configs c ON (l.fee_line_config_id = c.id)
WHERE type = 'flat_fee' AND status = 'detailed' AND l.parent_line_id = sqlc.arg(parent_line_id)::varchar;

-- name: GetUsageBasedLineByID :one

SELECT l.*, c.price_type, c.feature_key, c.price, c.pre_line_period_quantity, c.metered_quantity, c.metered_pre_line_period_quantity
FROM public.billing_invoice_lines l JOIN public.billing_invoice_usage_based_line_configs c ON (l.usage_based_line_config_id = c.id)
WHERE type = 'usage_based' AND status = 'valid' AND l.id = sqlc.arg(line_id)::varchar;
