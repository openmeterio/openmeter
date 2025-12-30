--- Restore config data.
UPDATE entitlements
SET config = to_jsonb(encode(convert_to(to_json(config::text)::text, 'UTF8'), 'base64')::text)
WHERE entitlement_type = 'static' AND config IS NOT NULL;
