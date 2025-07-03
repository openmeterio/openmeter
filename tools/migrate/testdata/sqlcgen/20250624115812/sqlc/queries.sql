-- Insert queries for test setup
-- name: CreateFeature :exec
INSERT INTO features (
    namespace,
    id,
    key,
    name,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: CreateEntitlement :exec
INSERT INTO entitlements (
    namespace,
    id,
    created_at,
    updated_at,
    entitlement_type,
    feature_key,
    feature_id,
    subject_key,
    usage_period_interval,
    usage_period_anchor
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
);

-- name: CreateUsageReset :exec
INSERT INTO usage_resets (
    namespace,
    id,
    created_at,
    updated_at,
    entitlement_id,
    reset_time,
    anchor
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- Query to verify usage reset data
-- name: GetUsageResetByID :one
SELECT * FROM usage_resets WHERE id = $1;

-- Query to get entitlement info
-- name: GetEntitlementByID :one
SELECT * FROM entitlements WHERE id = $1;

-- Placeholder query for SQLC validation
-- name: GetSchemaVersion :one
SELECT version FROM schema_om ORDER BY version DESC LIMIT 1;
