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
  "dimension_display_value" character varying NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "ledgerdimension_annotations" to table: "ledger_dimensions"
CREATE INDEX "ledgerdimension_annotations" ON "ledger_dimensions" USING gin ("annotations");
-- create index "ledgerdimension_id" to table: "ledger_dimensions"
CREATE UNIQUE INDEX "ledgerdimension_id" ON "ledger_dimensions" ("id");
-- create index "ledgerdimension_namespace" to table: "ledger_dimensions"
CREATE INDEX "ledgerdimension_namespace" ON "ledger_dimensions" ("namespace");
-- create index "ledgerdimension_namespace_dimension_key_dimension_value" to table: "ledger_dimensions"
CREATE UNIQUE INDEX "ledgerdimension_namespace_dimension_key_dimension_value" ON "ledger_dimensions" ("namespace", "dimension_key", "dimension_value");
-- create index "ledgerdimension_namespace_id" to table: "ledger_dimensions"
CREATE UNIQUE INDEX "ledgerdimension_namespace_id" ON "ledger_dimensions" ("namespace", "id");
-- create "ledger_sub_accounts" table
CREATE TABLE "ledger_sub_accounts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "annotations" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "account_id" character(26) NOT NULL,
  "ledger_dimension_sub_accounts" character(26) NULL,
  "currency_dimension_id" character(26) NOT NULL,
  "tax_code_dimension_id" character(26) NULL,
  "features_dimension_id" character(26) NULL,
  "credit_priority_dimension_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "ledger_sub_accounts_ledger_accounts_sub_accounts" FOREIGN KEY ("account_id") REFERENCES "ledger_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "ledger_sub_accounts_ledger_dimensions_credit_priority_sub_accou" FOREIGN KEY ("credit_priority_dimension_id") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "ledger_sub_accounts_ledger_dimensions_currency_sub_accounts" FOREIGN KEY ("currency_dimension_id") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "ledger_sub_accounts_ledger_dimensions_features_sub_accounts" FOREIGN KEY ("features_dimension_id") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "ledger_sub_accounts_ledger_dimensions_sub_accounts" FOREIGN KEY ("ledger_dimension_sub_accounts") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "ledger_sub_accounts_ledger_dimensions_tax_code_sub_accounts" FOREIGN KEY ("tax_code_dimension_id") REFERENCES "ledger_dimensions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- create index "ledgersubaccount_annotations" to table: "ledger_sub_accounts"
CREATE INDEX "ledgersubaccount_annotations" ON "ledger_sub_accounts" USING gin ("annotations");
-- create index "ledgersubaccount_id" to table: "ledger_sub_accounts"
CREATE UNIQUE INDEX "ledgersubaccount_id" ON "ledger_sub_accounts" ("id");
-- create index "ledgersubaccount_namespace" to table: "ledger_sub_accounts"
CREATE INDEX "ledgersubaccount_namespace" ON "ledger_sub_accounts" ("namespace");
-- create index "ledgersubaccount_namespace_account_id_currency_dimension_id" to table: "ledger_sub_accounts"
CREATE UNIQUE INDEX "ledgersubaccount_namespace_account_id_currency_dimension_id" ON "ledger_sub_accounts" ("namespace", "account_id", "currency_dimension_id");
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
  "amount" numeric NOT NULL,
  "sub_account_id" character(26) NOT NULL,
  "transaction_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "ledger_entries_ledger_sub_accounts_entries" FOREIGN KEY ("sub_account_id") REFERENCES "ledger_sub_accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
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
-- create index "ledgerentry_namespace_id" to table: "ledger_entries"
CREATE UNIQUE INDEX "ledgerentry_namespace_id" ON "ledger_entries" ("namespace", "id");
-- create index "ledgerentry_namespace_sub_account_id" to table: "ledger_entries"
CREATE INDEX "ledgerentry_namespace_sub_account_id" ON "ledger_entries" ("namespace", "sub_account_id");
-- create index "ledgerentry_namespace_transaction_id" to table: "ledger_entries"
CREATE INDEX "ledgerentry_namespace_transaction_id" ON "ledger_entries" ("namespace", "transaction_id");
