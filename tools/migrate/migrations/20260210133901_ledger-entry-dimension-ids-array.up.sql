-- create "ledger_accounts" table
CREATE TABLE "ledger_accounts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "account_type" character varying NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "ledgeraccount_annotations" to table: "ledger_accounts"
CREATE INDEX "ledgeraccount_annotations" ON "ledger_accounts" USING gin ("annotations");
-- create index "ledgeraccount_id" to table: "ledger_accounts"
CREATE UNIQUE INDEX "ledgeraccount_id" ON "ledger_accounts" ("id");
-- create index "ledgeraccount_namespace" to table: "ledger_accounts"
CREATE INDEX "ledgeraccount_namespace" ON "ledger_accounts" ("namespace");
-- create index "ledgeraccount_namespace_id" to table: "ledger_accounts"
CREATE UNIQUE INDEX "ledgeraccount_namespace_id" ON "ledger_accounts" ("namespace", "id");
-- create "ledger_dimensions" table
CREATE TABLE "ledger_dimensions" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "dimension_key" character varying NOT NULL,
  "dimension_value" character varying NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "ledgerdimension_annotations" to table: "ledger_dimensions"
CREATE INDEX "ledgerdimension_annotations" ON "ledger_dimensions" USING gin ("annotations");
-- create index "ledgerdimension_id" to table: "ledger_dimensions"
CREATE UNIQUE INDEX "ledgerdimension_id" ON "ledger_dimensions" ("id");
-- create index "ledgerdimension_namespace" to table: "ledger_dimensions"
CREATE INDEX "ledgerdimension_namespace" ON "ledger_dimensions" ("namespace");
-- create index "ledgerdimension_namespace_dimension_key_dimension_value" to table: "ledger_dimensions"
CREATE INDEX "ledgerdimension_namespace_dimension_key_dimension_value" ON "ledger_dimensions" ("namespace", "dimension_key", "dimension_value");
-- create index "ledgerdimension_namespace_id" to table: "ledger_dimensions"
CREATE UNIQUE INDEX "ledgerdimension_namespace_id" ON "ledger_dimensions" ("namespace", "id");
-- create "ledger_transaction_groups" table
CREATE TABLE "ledger_transaction_groups" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- create index "ledgertransactiongroup_annotations" to table: "ledger_transaction_groups"
CREATE INDEX "ledgertransactiongroup_annotations" ON "ledger_transaction_groups" USING gin ("annotations");
-- create index "ledgertransactiongroup_id" to table: "ledger_transaction_groups"
CREATE UNIQUE INDEX "ledgertransactiongroup_id" ON "ledger_transaction_groups" ("id");
-- create index "ledgertransactiongroup_namespace" to table: "ledger_transaction_groups"
CREATE INDEX "ledgertransactiongroup_namespace" ON "ledger_transaction_groups" ("namespace");
-- create index "ledgertransactiongroup_namespace_id" to table: "ledger_transaction_groups"
CREATE UNIQUE INDEX "ledgertransactiongroup_namespace_id" ON "ledger_transaction_groups" ("namespace", "id");
-- create "ledger_transactions" table
CREATE TABLE "ledger_transactions" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "booked_at" timestamptz NOT NULL,
  "group_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "ledger_transactions_ledger_transaction_groups_transactions" FOREIGN KEY ("group_id") REFERENCES "ledger_transaction_groups" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "ledgertransaction_annotations" to table: "ledger_transactions"
CREATE INDEX "ledgertransaction_annotations" ON "ledger_transactions" USING gin ("annotations");
-- create index "ledgertransaction_id" to table: "ledger_transactions"
CREATE UNIQUE INDEX "ledgertransaction_id" ON "ledger_transactions" ("id");
-- create index "ledgertransaction_namespace" to table: "ledger_transactions"
CREATE INDEX "ledgertransaction_namespace" ON "ledger_transactions" ("namespace");
-- create index "ledgertransaction_namespace_booked_at" to table: "ledger_transactions"
CREATE INDEX "ledgertransaction_namespace_booked_at" ON "ledger_transactions" ("namespace", "booked_at");
-- create index "ledgertransaction_namespace_group_id" to table: "ledger_transactions"
CREATE INDEX "ledgertransaction_namespace_group_id" ON "ledger_transactions" ("namespace", "group_id");
-- create index "ledgertransaction_namespace_id" to table: "ledger_transactions"
CREATE UNIQUE INDEX "ledgertransaction_namespace_id" ON "ledger_transactions" ("namespace", "id");
-- create "ledger_entries" table
CREATE TABLE "ledger_entries" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "account_id" character(26) NOT NULL,
  "account_type" character varying NOT NULL,
  "dimension_ids" text[] NULL,
  "amount" numeric NOT NULL,
  "transaction_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "ledger_entries_ledger_transactions_entries" FOREIGN KEY ("transaction_id") REFERENCES "ledger_transactions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "ledgerentry_annotations" to table: "ledger_entries"
CREATE INDEX "ledgerentry_annotations" ON "ledger_entries" USING gin ("annotations");
-- create index "ledgerentry_created_at_id" to table: "ledger_entries"
CREATE INDEX "ledgerentry_created_at_id" ON "ledger_entries" ("created_at", "id") WHERE (deleted_at IS NULL);
-- create index "ledgerentry_id" to table: "ledger_entries"
CREATE UNIQUE INDEX "ledgerentry_id" ON "ledger_entries" ("id");
-- create index "ledgerentry_namespace" to table: "ledger_entries"
CREATE INDEX "ledgerentry_namespace" ON "ledger_entries" ("namespace");
-- create index "ledgerentry_namespace_account_id" to table: "ledger_entries"
CREATE INDEX "ledgerentry_namespace_account_id" ON "ledger_entries" ("namespace", "account_id");
-- create index "ledgerentry_namespace_id" to table: "ledger_entries"
CREATE UNIQUE INDEX "ledgerentry_namespace_id" ON "ledger_entries" ("namespace", "id");
-- create index "ledgerentry_namespace_transaction_id" to table: "ledger_entries"
CREATE INDEX "ledgerentry_namespace_transaction_id" ON "ledger_entries" ("namespace", "transaction_id");
-- create function "validate_ledger_entry_dimension_ids"
CREATE FUNCTION "validate_ledger_entry_dimension_ids"() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.dimension_ids IS NULL OR array_length(NEW.dimension_ids, 1) IS NULL THEN
    RETURN NEW;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM unnest(NEW.dimension_ids) AS dim_id
    LEFT JOIN ledger_dimensions d
      ON d.id = dim_id
     AND d.namespace = NEW.namespace
    WHERE d.id IS NULL
  ) THEN
    RAISE EXCEPTION 'ledger entry references non-existent dimension id'
      USING ERRCODE = '23503',
            CONSTRAINT = 'ledger_entries_dimension_ids_fk';
  END IF;

  RETURN NEW;
END;
$$;
-- create trigger "ledger_entries_dimension_ids_fk" on table: "ledger_entries"
CREATE TRIGGER "ledger_entries_dimension_ids_fk"
BEFORE INSERT OR UPDATE OF "dimension_ids", "namespace" ON "ledger_entries"
FOR EACH ROW
EXECUTE FUNCTION "validate_ledger_entry_dimension_ids"();
