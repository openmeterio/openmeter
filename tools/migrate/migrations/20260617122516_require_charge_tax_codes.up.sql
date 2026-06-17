-- backfill charge tax_code_id values from namespace defaults before making the column required
-- This was already handled by the tax code backfills, but let's make sure we don't have any missing values.
UPDATE "charge_credit_purchases" AS c
SET "tax_code_id" = d."credit_grant_tax_code_id"
FROM "organization_default_tax_codes" AS d
WHERE c."tax_code_id" IS NULL
  AND c."namespace" = d."namespace"
  AND d."deleted_at" IS NULL;

UPDATE "charge_flat_fees" AS c
SET "tax_code_id" = d."invoicing_tax_code_id"
FROM "organization_default_tax_codes" AS d
WHERE c."tax_code_id" IS NULL
  AND c."namespace" = d."namespace"
  AND d."deleted_at" IS NULL;

UPDATE "charge_usage_based" AS c
SET "tax_code_id" = d."invoicing_tax_code_id"
FROM "organization_default_tax_codes" AS d
WHERE c."tax_code_id" IS NULL
  AND c."namespace" = d."namespace"
  AND d."deleted_at" IS NULL;

DO $$
DECLARE
  credit_purchase_missing bigint;
  flat_fee_missing bigint;
  usage_based_missing bigint;
BEGIN
  SELECT count(*) INTO credit_purchase_missing FROM "charge_credit_purchases" WHERE "tax_code_id" IS NULL;
  SELECT count(*) INTO flat_fee_missing FROM "charge_flat_fees" WHERE "tax_code_id" IS NULL;
  SELECT count(*) INTO usage_based_missing FROM "charge_usage_based" WHERE "tax_code_id" IS NULL;

  IF credit_purchase_missing > 0 OR flat_fee_missing > 0 OR usage_based_missing > 0 THEN
    RAISE EXCEPTION 'require_charge_tax_codes: missing tax_code_id after default backfill: credit_purchase=%, flat_fee=%, usage_based=%',
      credit_purchase_missing, flat_fee_missing, usage_based_missing;
  END IF;
END $$;

-- modify "charge_credit_purchases" table
-- atlas:nolint MF104
ALTER TABLE "charge_credit_purchases" DROP CONSTRAINT "charge_credit_purchases_tax_codes_charge_credit_purchases", ALTER COLUMN "tax_code_id" SET NOT NULL, ADD CONSTRAINT "charge_credit_purchases_tax_codes_charge_credit_purchases" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- modify "charge_flat_fees" table
-- atlas:nolint MF104
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_tax_codes_charge_flat_fees", ALTER COLUMN "tax_code_id" SET NOT NULL, ADD CONSTRAINT "charge_flat_fees_tax_codes_charge_flat_fees" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- modify "charge_usage_based" table
-- atlas:nolint MF104
ALTER TABLE "charge_usage_based" DROP CONSTRAINT "charge_usage_based_tax_codes_charge_usage_based", ALTER COLUMN "tax_code_id" SET NOT NULL, ADD CONSTRAINT "charge_usage_based_tax_codes_charge_usage_based" FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
