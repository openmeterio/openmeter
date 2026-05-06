-- Reverse routing_key backfill: strip the tax_behavior:... segment from V1 rows.
UPDATE "ledger_sub_account_routes"
SET "routing_key" = regexp_replace(
    "routing_key",
    '\|tax_behavior:[^|]*(\|features:)',
    '\1'
)
WHERE "routing_key_version" = 'v1'
  AND "routing_key" LIKE '%|tax_behavior:%';
-- reverse: modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" DROP COLUMN "tax_behavior";
