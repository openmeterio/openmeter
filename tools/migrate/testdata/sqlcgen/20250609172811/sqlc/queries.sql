-- name: GetSplitLineGroup :one
SELECT *
    FROM public.billing_invoice_split_line_groups
    WHERE id = $1;

-- name: GetUsageBasedLinesBySplitLineGroup :many
SELECT l.*, c.price, c.feature_key
    FROM public.billing_invoice_lines l JOIN public.billing_invoice_usage_based_line_configs c ON (l.usage_based_line_config_id = c.id)
    WHERE type = 'usage_based' AND status = 'valid' AND split_line_group_id = $1;

-- name: CountLinesByStatusType :many
SELECT status, type, count(*)
    FROM public.billing_invoice_lines
    GROUP BY status, type;
