-- reverse: rename a column from "skip_issuing_sync_hook" to "enable_issuing_sync_hook"
ALTER TABLE "app_custom_invoicings" RENAME COLUMN "enable_issuing_sync_hook" TO "skip_issuing_sync_hook";
UPDATE "app_custom_invoicings" SET "skip_issuing_sync_hook" = NOT "skip_issuing_sync_hook";

-- reverse: rename a column from "skip_draft_sync_hook" to "enable_draft_sync_hook"
ALTER TABLE "app_custom_invoicings" RENAME COLUMN "enable_draft_sync_hook" TO "skip_draft_sync_hook";
UPDATE "app_custom_invoicings" SET "skip_draft_sync_hook" = NOT "skip_draft_sync_hook";
