-- Restore uppercase states, including deliveries created after upgrade.
-- Stop and drain lowercase-state writers before rolling back to older readers.
-- Scheduling, attempts JSON and the column default stay unchanged.
UPDATE notification_event_delivery_status
SET state = CASE state
    WHEN 'pending' THEN 'PENDING'
    WHEN 'sending' THEN 'SENDING'
    WHEN 'resending' THEN 'RESENDING'
    WHEN 'success' THEN 'SUCCESS'
    WHEN 'failed' THEN 'FAILED'
END
WHERE state IN ('pending', 'sending', 'resending', 'success', 'failed');
