-- Breakage bookkeeping is inserted before the caller posts its ledger group.
-- References must exist when the enclosing transaction commits.
ALTER TABLE "ledger_breakage_records"
    ALTER CONSTRAINT "ledger_breakage_records_ledger_entries_source_breakage_records" DEFERRABLE INITIALLY DEFERRED,
    ALTER CONSTRAINT "ledger_breakage_records_ledger_transaction_groups_breakage_reco" DEFERRABLE INITIALLY DEFERRED,
    ALTER CONSTRAINT "ledger_breakage_records_ledger_transaction_groups_source_breaka" DEFERRABLE INITIALLY DEFERRED,
    ALTER CONSTRAINT "ledger_breakage_records_ledger_transactions_breakage_records" DEFERRABLE INITIALLY DEFERRED,
    ALTER CONSTRAINT "ledger_breakage_records_ledger_transactions_source_breakage_rec" DEFERRABLE INITIALLY DEFERRED;

-- Ent's implicit join table has a status-first primary key. Event reads look up
-- the status IDs in the opposite direction.
CREATE INDEX "notification_status_events_event_status_idx"
    ON "notification_event_delivery_status_events" ("notification_event_id", "notification_event_delivery_status_id");
