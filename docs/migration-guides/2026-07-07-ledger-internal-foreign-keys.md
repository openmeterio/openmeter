# Ledger Internal Foreign Key Validation

This migration adds foreign keys for ledger-owned references in `ledger_customer_accounts` and `ledger_breakage_records`.

The migration creates these constraints as `NOT VALID`. New writes are checked immediately, but existing rows are not scanned during the migration. This keeps the migration job short and lets operators validate historical rows separately.

## Migration Steps

First, apply OpenMeter migrations as usual.

After the migration has been applied, validate the new constraints manually:

```sql
SET lock_timeout = '5s';
SET statement_timeout = '0';

ALTER TABLE ledger_breakage_records VALIDATE CONSTRAINT ledger_breakage_records_ledger_breakage_records_planned_release;
ALTER TABLE ledger_breakage_records VALIDATE CONSTRAINT ledger_breakage_records_ledger_breakage_records_release_reopens;
ALTER TABLE ledger_breakage_records VALIDATE CONSTRAINT ledger_breakage_records_ledger_entries_source_breakage_records;
ALTER TABLE ledger_breakage_records VALIDATE CONSTRAINT ledger_breakage_records_ledger_sub_accounts_breakage_records;
ALTER TABLE ledger_breakage_records VALIDATE CONSTRAINT ledger_breakage_records_ledger_sub_accounts_fbo_breakage_record;
ALTER TABLE ledger_breakage_records VALIDATE CONSTRAINT ledger_breakage_records_ledger_transaction_groups_breakage_reco;
ALTER TABLE ledger_breakage_records VALIDATE CONSTRAINT ledger_breakage_records_ledger_transaction_groups_source_breaka;
ALTER TABLE ledger_breakage_records VALIDATE CONSTRAINT ledger_breakage_records_ledger_transactions_breakage_records;
ALTER TABLE ledger_breakage_records VALIDATE CONSTRAINT ledger_breakage_records_ledger_transactions_source_breakage_rec;

ALTER TABLE ledger_customer_accounts VALIDATE CONSTRAINT ledger_customer_accounts_ledger_accounts_customer_accounts;
```

For large databases, run these statements one at a time. If validation fails, the failed constraint identifies the relationship with orphaned historical rows. Fix those rows and rerun the failed validation statement.
