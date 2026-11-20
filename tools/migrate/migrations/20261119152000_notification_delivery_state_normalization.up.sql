-- Normalize delivery states after all backends and workers write lowercase
-- states and transactions started by older writers have completed.
-- Scheduling, attempts JSON and the column default stay unchanged.
UPDATE notification_event_delivery_status
SET state = CASE state
    WHEN 'PENDING' THEN 'pending'
    WHEN 'SENDING' THEN 'sending'
    WHEN 'RESENDING' THEN 'resending'
    WHEN 'SUCCESS' THEN 'success'
    WHEN 'FAILED' THEN 'failed'
END
WHERE state IN ('PENDING', 'SENDING', 'RESENDING', 'SUCCESS', 'FAILED');
