-- Post-migration queries with usage_period_interval column

-- Query to verify usage reset data (including the new column)
-- name: GetUsageResetByID :one
SELECT * FROM usage_resets WHERE id = $1;

-- Query to get usage_period_interval specifically
-- name: GetUsageResetInterval :one
SELECT usage_period_interval FROM usage_resets WHERE id = $1;

-- Query to get all usage resets for an entitlement
-- name: GetUsageResetsByEntitlementID :many
SELECT * FROM usage_resets WHERE entitlement_id = $1 ORDER BY reset_time ASC;

-- Query to get an entitlement by ID
-- name: GetEntitlementByID :one
SELECT * FROM entitlements WHERE id = $1;

-- Insert usage reset with the new column (should fail with NULL)
-- name: CreateUsageResetWithInterval :exec
INSERT INTO usage_resets (
    namespace,
    id,
    created_at,
    updated_at,
    entitlement_id,
    reset_time,
    anchor,
    usage_period_interval
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: CreateFeature :exec
INSERT INTO features (
    namespace,
    id,
    key,
    name,
    created_at,
    updated_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
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
    usage_period_anchor,
    measure_usage_from
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11
);


-- Placeholder query for SQLC validation
-- name: GetSchemaVersion :one
SELECT version FROM schema_om ORDER BY version DESC LIMIT 1;
