-- This migration consolidates RateCard information from planratecards, addonratecards, and subscriptionitems
-- into the central rate_cards table, while keeping the original data intact for safety.

-- Create a function to generate a ULID-compatible IDs
-- For details, see the canonical spec: https://github.com/ulid/spec
CREATE OR REPLACE FUNCTION om_generate_ulid() RETURNS char(26) AS $$
DECLARE
  -- Get the current timestamp in milliseconds (UNIX timestamp)
  timestamp_ms BIGINT := (EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) * 1000)::BIGINT;
  -- Convert to base32 encoding (using a subset of characters compatible with ULID)
  timestamp_base32 CHAR(10);
  -- Random part (16 characters)
  random_part CHAR(16);
  -- Character set for base32 encoding (excluding I, L, O, U to avoid confusion)
  chars CHAR(32) := '0123456789ABCDEFGHJKMNPQRSTVWXYZ';
  -- Temp variables
  remainder INT;
  i INT;
  random_val INT;
BEGIN
  -- Generate timestamp part (first 10 chars)
  timestamp_base32 := '';
  FOR i IN 1..10 LOOP
    remainder := timestamp_ms % 32;
    timestamp_ms := timestamp_ms / 32;
    timestamp_base32 := substr(chars, remainder+1, 1) || timestamp_base32;
  END LOOP;

  -- Generate random part (16 chars)
  random_part := '';
  FOR i IN 1..16 LOOP
    random_val := (random() * 31 + 1)::INT;
    random_part := random_part || substr(chars, random_val, 1);
  END LOOP;

  -- Combine timestamp and random parts
  RETURN timestamp_base32 || random_part;
END;
$$ LANGUAGE plpgsql;

-- First, migrate planratecards without a ratecard_id reference yet
WITH inserted_ratecards AS (
    INSERT INTO rate_cards (
        id,
        namespace,
        metadata,
        created_at,
        updated_at,
        deleted_at,
        name,
        description,
        key,
        entitlement_template_entitlement_type,
        entitlement_template_metadata,
        entitlement_template_is_soft_limit,
        entitlement_template_issue_after_reset,
        entitlement_template_issue_after_reset_priority,
        entitlement_template_preserve_overage_at_reset,
        entitlement_template_config,
        entitlement_template_usage_period,
        type,
        feature_key,
        feature_id,
        tax_config,
        billing_cadence,
        price,
        discounts
    )
    SELECT
        om_generate_ulid() as id,
        plan_rate_cards.namespace,
        -- As we can only return fields of the target table, to be able to join in the next step we add the original id as a metadata field
        COALESCE(plan_rate_cards.metadata, '{}'::jsonb) || jsonb_build_object('sql_migration_original_plan_rate_card_id', plan_rate_cards.id) as metadata,
        plan_rate_cards.created_at,
        plan_rate_cards.updated_at,
        plan_rate_cards.deleted_at,
        plan_rate_cards.name,
        plan_rate_cards.description,
        plan_rate_cards.key,
        CASE
            WHEN (plan_rate_cards.entitlement_template->>'entitlementType') = 'metered' THEN 'metered'
            WHEN (plan_rate_cards.entitlement_template->>'entitlementType') = 'static' THEN 'static'
            WHEN (plan_rate_cards.entitlement_template->>'entitlementType') = 'boolean' THEN 'boolean'
            ELSE 'metered'
        END as entitlement_template_entitlement_type,
        COALESCE((plan_rate_cards.entitlement_template->>'metadata')::jsonb, '{}'::jsonb) as entitlement_template_metadata,
        (plan_rate_cards.entitlement_template->>'isSoftLimit')::boolean as entitlement_template_is_soft_limit,
        (plan_rate_cards.entitlement_template->>'issueAfterReset')::float as entitlement_template_issue_after_reset,
        (plan_rate_cards.entitlement_template->>'issueAfterResetPriority')::smallint as entitlement_template_issue_after_reset_priority,
        (plan_rate_cards.entitlement_template->>'preserveOverageAtReset')::boolean as entitlement_template_preserve_overage_at_reset,
        COALESCE((plan_rate_cards.entitlement_template->>'config')::jsonb, '{}'::jsonb) as entitlement_template_config,
        (plan_rate_cards.entitlement_template->>'usagePeriod') as entitlement_template_usage_period,
        plan_rate_cards.type,
        plan_rate_cards.feature_key,
        plan_rate_cards.feature_id,
        plan_rate_cards.tax_config,
        plan_rate_cards.billing_cadence,
        plan_rate_cards.price,
        plan_rate_cards.discounts
    FROM plan_rate_cards
    WHERE plan_rate_cards.ratecard_id IS NULL
    RETURNING id, (metadata->>'sql_migration_original_plan_rate_card_id')::text as original_id
)
UPDATE plan_rate_cards
SET ratecard_id = inserted_ratecards.id
FROM inserted_ratecards
WHERE plan_rate_cards.id = inserted_ratecards.original_id;

-- Next, migrate addonratecards without a ratecard_id reference yet
WITH inserted_ratecards AS (
    INSERT INTO rate_cards (
        id,
        namespace,
        metadata,
        created_at,
        updated_at,
        deleted_at,
        name,
        description,
        key,
        entitlement_template_entitlement_type,
        entitlement_template_metadata,
        entitlement_template_is_soft_limit,
        entitlement_template_issue_after_reset,
        entitlement_template_issue_after_reset_priority,
        entitlement_template_preserve_overage_at_reset,
        entitlement_template_config,
        entitlement_template_usage_period,
        type,
        feature_key,
        feature_id,
        tax_config,
        billing_cadence,
        price,
        discounts
    )
    SELECT
        om_generate_ulid() as id,
        addon_rate_cards.namespace,
        -- As we can only return fields of the target table, to be able to join in the next step we add the original id as a metadata field
        COALESCE(addon_rate_cards.metadata, '{}'::jsonb) || jsonb_build_object('sql_migration_original_addon_rate_card_id', addon_rate_cards.id) as metadata,
        addon_rate_cards.created_at,
        addon_rate_cards.updated_at,
        addon_rate_cards.deleted_at,
        addon_rate_cards.name,
        addon_rate_cards.description,
        addon_rate_cards.key,
        CASE
            WHEN (addon_rate_cards.entitlement_template->>'entitlementType') = 'metered' THEN 'metered'
            WHEN (addon_rate_cards.entitlement_template->>'entitlementType') = 'static' THEN 'static'
            WHEN (addon_rate_cards.entitlement_template->>'entitlementType') = 'boolean' THEN 'boolean'
            ELSE 'metered'
        END as entitlement_template_entitlement_type,
        COALESCE((addon_rate_cards.entitlement_template->>'metadata')::jsonb, '{}'::jsonb) as entitlement_template_metadata,
        (addon_rate_cards.entitlement_template->>'isSoftLimit')::boolean as entitlement_template_is_soft_limit,
        (addon_rate_cards.entitlement_template->>'issueAfterReset')::float as entitlement_template_issue_after_reset,
        (addon_rate_cards.entitlement_template->>'issueAfterResetPriority')::smallint as entitlement_template_issue_after_reset_priority,
        (addon_rate_cards.entitlement_template->>'preserveOverageAtReset')::boolean as entitlement_template_preserve_overage_at_reset,
        COALESCE((addon_rate_cards.entitlement_template->>'config')::jsonb, '{}'::jsonb) as entitlement_template_config,
        (addon_rate_cards.entitlement_template->>'usagePeriod') as entitlement_template_usage_period,
        addon_rate_cards.type,
        addon_rate_cards.feature_key,
        addon_rate_cards.feature_id,
        addon_rate_cards.tax_config,
        addon_rate_cards.billing_cadence,
        addon_rate_cards.price,
        addon_rate_cards.discounts
    FROM addon_rate_cards
    WHERE addon_rate_cards.ratecard_id IS NULL
    RETURNING id, (metadata->>'sql_migration_original_addon_rate_card_id')::text as original_id
)
UPDATE addon_rate_cards
SET ratecard_id = inserted_ratecards.id
FROM inserted_ratecards
WHERE addon_rate_cards.id = inserted_ratecards.original_id;

-- Finally, migrate subscription_items without a ratecard_id reference yet
WITH inserted_ratecards AS (
    INSERT INTO rate_cards (
        id,
        namespace,
        metadata,
        created_at,
        updated_at,
        deleted_at,
        name,
        description,
        key,
        entitlement_template_entitlement_type,
        entitlement_template_metadata,
        entitlement_template_is_soft_limit,
        entitlement_template_issue_after_reset,
        entitlement_template_issue_after_reset_priority,
        entitlement_template_preserve_overage_at_reset,
        entitlement_template_config,
        entitlement_template_usage_period,
        type,
        feature_key,
        feature_id,
        tax_config,
        billing_cadence,
        price,
        discounts
    )
    SELECT
        om_generate_ulid() as id,
        subscription_items.namespace,
        -- As we can only return fields of the target table, to be able to join in the next step we add the original id as a metadata field
        COALESCE(subscription_items.metadata, '{}'::jsonb) || jsonb_build_object('sql_migration_original_subscription_item_id', subscription_items.id) as metadata,
        subscription_items.created_at,
        subscription_items.updated_at,
        subscription_items.deleted_at,
        subscription_items.name,
        subscription_items.description,
        subscription_items.key,
        CASE
            WHEN (subscription_items.entitlement_template->>'entitlementType') = 'metered' THEN 'metered'
            WHEN (subscription_items.entitlement_template->>'entitlementType') = 'static' THEN 'static'
            WHEN (subscription_items.entitlement_template->>'entitlementType') = 'boolean' THEN 'boolean'
            ELSE 'metered'
        END as entitlement_template_entitlement_type,
        COALESCE((subscription_items.entitlement_template->>'metadata')::jsonb, '{}'::jsonb) as entitlement_template_metadata,
        (subscription_items.entitlement_template->>'isSoftLimit')::boolean as entitlement_template_is_soft_limit,
        (subscription_items.entitlement_template->>'issueAfterReset')::float as entitlement_template_issue_after_reset,
        (subscription_items.entitlement_template->>'issueAfterResetPriority')::smallint as entitlement_template_issue_after_reset_priority,
        (subscription_items.entitlement_template->>'preserveOverageAtReset')::boolean as entitlement_template_preserve_overage_at_reset,
        COALESCE((subscription_items.entitlement_template->>'config')::jsonb, '{}'::jsonb) as entitlement_template_config,
        (subscription_items.entitlement_template->>'usagePeriod') as entitlement_template_usage_period,
        'usage_based', -- Assuming this is the default type for subscription items
        subscription_items.feature_key,
        COALESCE(entitlement_feature.id, feature_rank.id, NULL) as feature_id,
        subscription_items.tax_config,
        subscription_items.billing_cadence,
        subscription_items.price,
        subscription_items.discounts
    FROM subscription_items
    LEFT JOIN entitlements ON subscription_items.entitlement_id = entitlements.id
    LEFT JOIN features entitlement_feature ON entitlements.feature_id = entitlement_feature.id
    LEFT JOIN (
        SELECT id, key, archived_at, RANK() OVER (PARTITION BY namespace, key ORDER BY created_at DESC) as feature_active_rank
        FROM features
    ) AS feature_rank ON
        subscription_items.feature_key = feature_rank.key AND feature_rank.feature_active_rank = 1 AND (
            feature_rank.archived_at IS NULL OR feature_rank.archived_at > subscription_items.created_at
        )
    WHERE subscription_items.ratecard_id IS NULL
    RETURNING id, (metadata->>'sql_migration_original_subscription_item_id')::text as original_id
)
UPDATE subscription_items
SET ratecard_id = inserted_ratecards.id
FROM inserted_ratecards
WHERE subscription_items.id = inserted_ratecards.original_id;

-- Clean up the function after we're done
DROP FUNCTION IF EXISTS om_generate_ulid();
