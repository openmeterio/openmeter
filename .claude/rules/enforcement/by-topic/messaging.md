# Enforcement: messaging (3 rules)

Topic file. Loaded on demand when an agent works on something in the `messaging` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Decision Violations (block)

### `dec-eventbus-001` — Publish domain events through eventbus.Publisher and consume via grouphandler, not direct confluent-kafka calls

*source: `deep_scan`*

**Why:** Outbound: eventbus.New wraps a watermill cqrs.EventBus and routes each event to one of three Kafka topics by event-name prefix (ingest → IngestEventsTopic, balance-worker → BalanceWorkerEventsTopic, everything else → SystemEventsTopic). Inbound: grouphandler.NewNoPublishingHandler builds map[eventName][]GroupEventHandler, derives the CloudEvent type per message, unmarshals once, runs matching handlers, and counts unknown types as ignored. Direct confluent-kafka-go produce/consume in a worker bypasses the centralized marshaling, topic routing, and type dispatch, and a single topic for all events breaks the throughput/retention separation.

## Pattern Divergence (inform)

### `sem-webhook-001` — Deliver notification webhooks behind the webhook.Handler interface with Svix and noop implementations

*source: `deep_scan`*

**Why:** notification/webhook defines a Handler interface (CreateWebhook, UpdateWebhook, endpoint secret/header management) with a Svix implementation (webhook/svix) and a noop implementation (webhook/noop) for when webhooks are disabled. Svix calls are wrapped in OpenTelemetry tracex spans; the event pipeline reconciles delivery state via eventhandler/reconcile.go. New delivery behavior must go behind the Handler interface so the noop swap keeps working when webhooks are off.

### `sem-grouphandler-001` — Build Kafka consumers with grouphandler type-routed dispatch; ack unknown event types as ignored

*source: `deep_scan`*

**Why:** grouphandler.NewNoPublishingHandler builds a map[eventName][]GroupEventHandler. On each message it derives the CloudEvent type, looks up handlers, unmarshals once into handler[0].NewEvent(), and runs all matching handlers joining their errors. Unknown event types are counted as 'ignored' and ack'd (return nil), not failed; per-message processing time and status counters are emitted to OpenTelemetry. New worker consumers should register handlers into this map rather than hand-rolling consume loops.
