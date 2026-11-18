-- Deploy the Phase 2 writers everywhere and drain legacy writers before applying.
UPDATE "notification_channels"
SET "type" = 'webhook'
WHERE "type" = 'WEBHOOK';
