-- rename a column from "skip_draft_sync_hook" to "enable_draft_sync_hook"
-- atlas:nolint BC102
ALTER TABLE "app_custom_invoicings" RENAME COLUMN "skip_draft_sync_hook" TO "enable_draft_sync_hook";
UPDATE "app_custom_invoicings" SET "enable_draft_sync_hook" = NOT "enable_draft_sync_hook";
-- rename a column from "skip_issuing_sync_hook" to "enable_issuing_sync_hook"
-- atlas:nolint BC102
ALTER TABLE "app_custom_invoicings" RENAME COLUMN "skip_issuing_sync_hook" TO "enable_issuing_sync_hook";
UPDATE "app_custom_invoicings" SET "enable_issuing_sync_hook" = NOT "enable_issuing_sync_hook";
