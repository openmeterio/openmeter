-- Repair migrations that may have been skipped on production databases because
-- their version numbers were lower than migrations already applied after a rebase.
--
-- This intentionally excludes 20260326000000_llmcost_normalize_providers.

-- 20260428102826_add_tax_codes_to_charges
ALTER TABLE "charge_credit_purchases"
  ADD COLUMN IF NOT EXISTS "tax_behavior" character varying NULL,
  ADD COLUMN IF NOT EXISTS "tax_code_id" character(26) NULL;

ALTER TABLE "charge_flat_fees"
  ADD COLUMN IF NOT EXISTS "tax_behavior" character varying NULL,
  ADD COLUMN IF NOT EXISTS "tax_code_id" character(26) NULL;

ALTER TABLE "charge_usage_based"
  ADD COLUMN IF NOT EXISTS "tax_behavior" character varying NULL,
  ADD COLUMN IF NOT EXISTS "tax_code_id" character(26) NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'charge_credit_purchases_tax_codes_charge_credit_purchases'
      AND conrelid = 'charge_credit_purchases'::regclass
  ) THEN
    ALTER TABLE "charge_credit_purchases"
      ADD CONSTRAINT "charge_credit_purchases_tax_codes_charge_credit_purchases"
      FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'charge_flat_fees_tax_codes_charge_flat_fees'
      AND conrelid = 'charge_flat_fees'::regclass
  ) THEN
    ALTER TABLE "charge_flat_fees"
      ADD CONSTRAINT "charge_flat_fees_tax_codes_charge_flat_fees"
      FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'charge_usage_based_tax_codes_charge_usage_based'
      AND conrelid = 'charge_usage_based'::regclass
  ) THEN
    ALTER TABLE "charge_usage_based"
      ADD CONSTRAINT "charge_usage_based_tax_codes_charge_usage_based"
      FOREIGN KEY ("tax_code_id") REFERENCES "tax_codes" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS "chargecreditpurchases_tax_code_id" ON "charge_credit_purchases" ("tax_code_id");
CREATE INDEX IF NOT EXISTS "chargeflatfees_tax_code_id" ON "charge_flat_fees" ("tax_code_id");
CREATE INDEX IF NOT EXISTS "chargeusagebased_tax_code_id" ON "charge_usage_based" ("tax_code_id");

DROP VIEW IF EXISTS "charges_search_v1s";
CREATE VIEW "charges_search_v1s" AS
SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", 'credit_purchase' AS "type" FROM "charge_credit_purchases" UNION ALL SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", 'flat_fee' AS "type" FROM "charge_flat_fees" UNION ALL SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", 'usage_based' AS "type" FROM "charge_usage_based";

-- 20260513083018_add_flat_fee_run_immutable
ALTER TABLE "charge_flat_fee_runs" ADD COLUMN IF NOT EXISTS "immutable" boolean NULL;

UPDATE "charge_flat_fee_runs"
SET "immutable" = false
WHERE "immutable" IS NULL;

ALTER TABLE "charge_flat_fee_runs" ALTER COLUMN "immutable" SET NOT NULL;

-- 20260514084134_add_ledger_entry_identity_key
ALTER TABLE "ledger_entries" ADD COLUMN IF NOT EXISTS "identity_key" character varying NOT NULL DEFAULT '';

UPDATE "ledger_entries"
SET "identity_key" = ''
WHERE "identity_key" IS NULL;

ALTER TABLE "ledger_entries"
  ALTER COLUMN "identity_key" SET DEFAULT '',
  ALTER COLUMN "identity_key" SET NOT NULL;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM "ledger_entries"
    GROUP BY "transaction_id", "sub_account_id", "identity_key"
    HAVING count(*) > 1
  ) THEN
    RAISE EXCEPTION 'cannot create ledgerentry_transaction_id_sub_account_id_identity_key: duplicate ledger_entries exist for (transaction_id, sub_account_id, identity_key)';
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS "ledgerentry_transaction_id_sub_account_id_identity_key" ON "ledger_entries" ("transaction_id", "sub_account_id", "identity_key");

-- 20260517121831_add_credit_expiration_breakage
ALTER TABLE "charge_credit_purchases" ADD COLUMN IF NOT EXISTS "expires_at" timestamptz NULL;

CREATE TABLE IF NOT EXISTS "ledger_breakage_records" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "kind" character varying NOT NULL,
  "amount" numeric NOT NULL,
  "customer_id" character(26) NOT NULL,
  "currency" character varying(3) NOT NULL,
  "credit_priority" bigint NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "source_kind" character varying NOT NULL,
  "source_transaction_group_id" character(26) NULL,
  "source_transaction_id" character(26) NULL,
  "source_entry_id" character(26) NULL,
  "breakage_transaction_group_id" character(26) NOT NULL,
  "breakage_transaction_id" character(26) NOT NULL,
  "fbo_sub_account_id" character(26) NOT NULL,
  "breakage_sub_account_id" character(26) NOT NULL,
  "plan_id" character(26) NULL,
  "release_id" character(26) NULL,
  PRIMARY KEY ("id")
);

CREATE INDEX IF NOT EXISTS "ledgerbreakagerecord_annotations" ON "ledger_breakage_records" USING gin ("annotations");
CREATE UNIQUE INDEX IF NOT EXISTS "ledgerbreakagerecord_id" ON "ledger_breakage_records" ("id");
CREATE INDEX IF NOT EXISTS "ledgerbreakagerecord_namespace" ON "ledger_breakage_records" ("namespace");
CREATE INDEX IF NOT EXISTS "ledgerbreakagerecord_namespace_breakage_transaction_group_id" ON "ledger_breakage_records" ("namespace", "breakage_transaction_group_id");
CREATE INDEX IF NOT EXISTS "ledgerbreakagerecord_namespace_customer_id_currency_credit_" ON "ledger_breakage_records" ("namespace", "customer_id", "currency", "credit_priority", "expires_at", "id");
CREATE INDEX IF NOT EXISTS "ledgerbreakagerecord_namespace_plan_id" ON "ledger_breakage_records" ("namespace", "plan_id");
CREATE INDEX IF NOT EXISTS "ledgerbreakagerecord_namespace_source_entry_id" ON "ledger_breakage_records" ("namespace", "source_entry_id");
CREATE INDEX IF NOT EXISTS "ledgerbreakagerecord_namespace_source_transaction_group_id" ON "ledger_breakage_records" ("namespace", "source_transaction_group_id");
