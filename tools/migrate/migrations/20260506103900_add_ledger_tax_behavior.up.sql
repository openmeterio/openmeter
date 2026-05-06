-- modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" ADD COLUMN "tax_behavior" character varying NULL;
-- Backfill routing_key for existing V1 rows: insert tax_behavior:null segment between tax_code and features.
-- Pre-existing rows were written without tax_behavior in the key; without this they would not be found by new lookups.
UPDATE "ledger_sub_account_routes"
SET "routing_key" = regexp_replace(
    "routing_key",
    '(\|tax_code:[^|]*)(\|features:)',
    '\1|tax_behavior:null\2'
)
WHERE "routing_key_version" = 'v1'
  AND "routing_key" NOT LIKE '%|tax_behavior:%';
