-- Normalize delivery states after all backends and workers write lowercase
-- states and transactions started by older writers have completed.
-- Scheduling, attempts JSON and the column default stay unchanged.
UPDATE notification_event_delivery_status
SET state = lower(state)
WHERE state IN ('PENDING', 'SENDING', 'RESENDING', 'SUCCESS', 'FAILED');
