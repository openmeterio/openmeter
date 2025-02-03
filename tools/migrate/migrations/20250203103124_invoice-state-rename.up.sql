UPDATE billing_invoices SET status = 'draft.created' WHERE status = 'draft_created';
UPDATE billing_invoices SET status = 'draft.updating' WHERE status = 'draft_updating';
UPDATE billing_invoices SET status = 'draft.manual_approval_needed' WHERE status = 'draft_manual_approval_needed';
UPDATE billing_invoices SET status = 'draft.validating' WHERE status = 'draft_validating';
UPDATE billing_invoices SET status = 'draft.invalid' WHERE status = 'draft_invalid';
UPDATE billing_invoices SET status = 'draft.syncing' WHERE status = 'draft_syncing';
UPDATE billing_invoices SET status = 'draft.sync_failed' WHERE status = 'draft_sync_failed';
UPDATE billing_invoices SET status = 'draft.waiting_auto_approval' WHERE status = 'draft_waiting_auto_approval';
UPDATE billing_invoices SET status = 'draft.ready_to_issue' WHERE status = 'draft_ready_to_issue';

UPDATE billing_invoices SET status = 'delete.in_progress' WHERE status = 'delete_in_progress';
UPDATE billing_invoices SET status = 'delete.syncing' WHERE status = 'delete_syncing';
UPDATE billing_invoices SET status = 'delete.failed' WHERE status = 'delete_failed';

UPDATE billing_invoices SET status = 'issuing.syncing' WHERE status = 'issuing_syncing';
UPDATE billing_invoices SET status = 'issuing.failed' WHERE status = 'issuing_failed';

