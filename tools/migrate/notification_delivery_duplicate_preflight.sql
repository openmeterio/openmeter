-- Each row identifies a destination whose recorded delivery histories need review.
SELECT namespace, event_id, channel_id, count(*) AS delivery_count,
       array_agg(id ORDER BY created_at, id) AS delivery_ids
FROM notification_event_delivery_status
GROUP BY namespace, event_id, channel_id
HAVING count(*) > 1
ORDER BY namespace, event_id, channel_id;
