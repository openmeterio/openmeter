-- Restore the legacy envelope from the current config, including Channels created
-- or updated after the upgrade.
-- Stop and drain writers of flat configs before rolling back to older readers.
UPDATE notification_channels
SET config = jsonb_build_object(
    'type', 'WEBHOOK',
    'data', (config - 'custom_headers' - 'signing_secret')
        || CASE WHEN config ? 'custom_headers'
            THEN jsonb_build_object('customHeaders', config->'custom_headers') ELSE '{}'::jsonb END
        || CASE WHEN config ? 'signing_secret'
            THEN jsonb_build_object('signingSecret', config->'signing_secret') ELSE '{}'::jsonb END
)
WHERE type IN ('WEBHOOK', 'webhook')
  AND jsonb_typeof(config) = 'object'
  AND NOT (config ?| ARRAY['type', 'data']);
