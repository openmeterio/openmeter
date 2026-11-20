-- Flatten legacy webhook Channel configs and retain the original JSON in annotations.
-- Apply after all backend and worker instances write flat configs and transactions
-- started by older instances have completed. Conversion is atomic: an unsupported
-- envelope aborts the migration without changing any Channels.
DO $$
DECLARE
    channel RECORD;
    data JSONB;
BEGIN
    FOR channel IN
        SELECT id, namespace, config
        FROM notification_channels
        WHERE type IN ('WEBHOOK', 'webhook')
          AND config ?| ARRAY['type', 'data']
        ORDER BY id
        FOR UPDATE
    LOOP
        IF (jsonb_typeof(channel.config) = 'object'
            AND channel.config->>'type' IN ('WEBHOOK', 'webhook')
            AND jsonb_typeof(channel.config->'data') = 'object') IS NOT TRUE THEN
            RAISE EXCEPTION 'invalid channel config envelope [table=notification_channels namespace=% channel.id=%]: expected type WEBHOOK/webhook and an object-valued data field. '
                'Repair this row''s config to contain that envelope with url and optional customHeaders/signingSecret inside data, or use a flat config with url and optional custom_headers/signing_secret. Preserve the existing URL, headers and signing secret. '
                'If schema_om shows version 20261119151800 as dirty, verify this migration rolled back and use golang-migrate with x-migrations-table=schema_om to force 20261119151700 (the previous successful version), then rerun the migration.', channel.namespace, channel.id;
        END IF;

        data := channel.config->'data';

        -- Config contents are application-owned. Only the storage shape changes;
        -- preserve absent, null and empty values, and keep header names unchanged.
        UPDATE notification_channels
        SET config = (data - 'customHeaders' - 'signingSecret')
            || CASE WHEN data ? 'customHeaders'
                THEN jsonb_build_object('custom_headers', data->'customHeaders') ELSE '{}'::jsonb END
            || CASE WHEN data ? 'signingSecret'
                THEN jsonb_build_object('signing_secret', data->'signingSecret') ELSE '{}'::jsonb END,
            -- Existing annotations take precedence so a rerun cannot replace the original backup.
            annotations = jsonb_build_object(
                'dbmigration.notification.channel.config.normalization.original', channel.config
            ) || COALESCE(NULLIF(annotations, 'null'::jsonb), '{}'::jsonb)
        WHERE id = channel.id AND namespace = channel.namespace;
    END LOOP;
END $$;
