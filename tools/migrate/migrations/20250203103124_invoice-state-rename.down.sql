UPDATE billing_invoices SET status = 'draft_created' WHERE status = 'draft.created';
UPDATE billing_invoices SET status = 'draft_updating' WHERE status = 'draft.updating';
UPDATE billing_invoices SET status = 'draft_manual_approval_needed' WHERE status = 'draft.manual_approval_needed';
UPDATE billing_invoices SET status = 'draft_validating' WHERE status = 'draft.validating';
UPDATE billing_invoices SET status = 'draft_invalid' WHERE status = 'draft.invalid';
UPDATE billing_invoices SET status = 'draft_syncing' WHERE status = 'draft.syncing';
UPDATE billing_invoices SET status = 'draft_sync_failed' WHERE status = 'draft.sync_failed';
UPDATE billing_invoices SET status = 'draft_waiting_auto_approval' WHERE status = 'draft.waiting_auto_approval';
UPDATE billing_invoices SET status = 'draft_ready_to_issue' WHERE status = 'draft.ready_to_issue';

UPDATE billing_invoices SET status = 'delete_in_progress' WHERE status = 'delete.in_progress';
UPDATE billing_invoices SET status = 'delete_syncing' WHERE status = 'delete.syncing';
UPDATE billing_invoices SET status = 'delete_failed' WHERE status = 'delete.failed';

UPDATE billing_invoices SET status = 'issuing_syncing' WHERE status = 'issuing_syncing';
UPDATE billing_invoices SET status = 'issuing_failed' WHERE status = 'issuing_failed';

