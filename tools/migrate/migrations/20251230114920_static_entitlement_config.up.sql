--- Update JSON configs to store JSON objects instead of their base64 encoded JSON string representation.
UPDATE entitlements
SET config = ((convert_from(decode((config #>> '{}')::text, 'base64'), 'UTF8'))::jsonb #>> '{}')::jsonb
WHERE entitlement_type = 'static' AND config IS NOT NULL;
