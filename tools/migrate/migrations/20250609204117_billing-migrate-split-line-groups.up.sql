
-- Step 1: convert existing split lines into split line groups

INSERT INTO public.billing_invoice_split_line_groups
(
    id,
    namespace,
    metadata,
    created_at,
    updated_at,
    deleted_at,
    name,
    description,
    service_period_start,
    service_period_end,
    currency,
    tax_config,
    unique_reference_id,
    ratecard_discounts,
    feature_key,
    price,
    subscription_id,
    subscription_phase_id,
    subscription_item_id
)
    SELECT
      l.id,
      l.namespace,
      l.metadata,
      l.created_at,
      l.updated_at,
      l.deleted_at,
      l.name,
      l.description,
      l.period_start,
      l.period_end,
      l.currency,
      l.tax_config,
      l.child_unique_reference_id,
      l.ratecard_discounts,
      u.feature_key,
      u.price,
      l.subscription_id,
      l.subscription_phase_id,
      l.subscription_item_id
    FROM
        public.billing_invoice_lines l JOIN public.billing_invoice_usage_based_line_configs u ON (l.usage_based_line_config_id = u.id)
    WHERE
        l.type = 'usage_based' AND l.status = 'split';

-- Step 2: Associate existing lines referencing the line lines to the split line group

UPDATE public.billing_invoice_lines
SET split_line_group_id = parent_line_id, parent_line_id = NULL
WHERE "type" = 'usage_based' and "status" = 'valid' and "parent_line_id" IS NOT NULL;

-- Step 3: delete the split lines

DELETE FROM public.billing_invoice_lines
WHERE "type" = 'usage_based' and "status" = 'split';
