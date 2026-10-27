# Kafka event subscriptions

New consumer groups keep Kafka's newest-offset policy. Messages published before
their partition consumers select starting offsets can be skipped.

The billing, balance, and notification workers expose `GET /healthz/subscriptions`
on their telemetry ports. It returns 503 until every configured topic has a
consumer-group session and every partition assigned
to this replica has entered `ConsumeClaim`, after Sarama selects its starting
offset. It returns 200 even on an empty topic or when a joined replica has no
assigned partitions. The JSON body contains `ready`, `assigned`, and `initialized`.

The status becomes unready on session cancellation, claim loss, or subscriber
shutdown. It reports consumer initialization, not backlog completion or successful
event processing. E2E startup waits for this endpoint before creating fixtures.

This status is separate from `/healthz`, `/healthz/live`, and `/healthz/ready`.
The existing Kubernetes health checks retain their behavior. The balance worker
waits for all three of its event-topic subscriptions, including idle sessions with
no assigned partitions.
