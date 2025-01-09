-- In all honesty we don't have much of a chance here in retrofitting the values. What we can do is keep them as is and use the deleted_at flag so the system will ignore them going on.
UPDATE
    entitlements
SET
    deleted_at = NOW();

UPDATE
    grants
SET
    deleted_at = NOW();