# Architecture

OpenMeter separates high-volume usage data from transactional application state.
The following diagram shows the primary runtime components and data flows; it omits observability and deployment-specific infrastructure.
The diagram requests the ELK layout engine; renderers without it installed (such as github.com) fall back to the default dagre layout with the same content.

```mermaid
---
config:
  layout: elk
---
flowchart LR
    subgraph clients["API consumers"]
        producers["Applications and SDKs<br/>producing usage events"]
        operators["Dashboard, portal, and<br/>management API clients"]
        webhookConsumers["Webhook consumers"]
    end

    subgraph runtime["OpenMeter runtime"]
        api["API server<br/>ingest, management, queries,<br/>and webhook delivery"]
        sink["Sink worker<br/>validate, deduplicate, and persist usage"]
        balance["Balance worker<br/>entitlements and credit balances"]
        billing["Billing worker<br/>subscription sync, rating,<br/>and invoice lifecycle"]
        notifications["Notification service<br/>rule evaluation and<br/>notification events"]
        jobs["Scheduled jobs<br/>migrations and cross-domain<br/>maintenance"]
    end

    subgraph data["Messaging and state"]
        kafka[("Kafka<br/>usage and domain events")]
        redis[("Redis<br/>optional deduplication and<br/>query progress state")]
        clickhouse[("ClickHouse<br/>usage events and aggregates")]
        postgres[("PostgreSQL<br/>customers, catalog, subscriptions,<br/>billing, and entitlements")]
        svix["Svix<br/>webhook delivery"]
    end

    producers -->|"CloudEvents"| api
    operators -->|"commands and queries"| api
    api -->|"publish events"| kafka
    api -->|"query usage"| clickhouse
    api <-->|"transactional state"| postgres
    api -.->|"ingest dedup and query progress"| redis
    api -->|"deliver webhooks and manage endpoints"| svix

    kafka -->|"usage events"| sink
    sink -.->|"deduplication keys"| redis
    sink -->|"read meter definitions"| postgres
    sink -->|"validated usage"| clickhouse
    sink -->|"ingest notifications"| kafka

    kafka -->|"ingest notifications and domain events"| balance
    balance -->|"balance snapshot events"| kafka
    balance -->|"query usage"| clickhouse
    balance <-->|"entitlement state and snapshot cache"| postgres

    kafka -->|"subscription and billing events"| billing
    billing -->|"invoice events"| kafka
    billing -->|"query usage"| clickhouse
    billing <-->|"charges and invoices"| postgres

    jobs -->|"maintenance reads and writes"| postgres
    jobs -->|"query usage"| clickhouse
    jobs -->|"recalculation and advance events"| kafka

    kafka -->|"balance and invoice events"| notifications
    notifications <-->|"rules and pending deliveries"| postgres
    svix -->|"signed webhooks"| webhookConsumers
```

Kafka decouples event ingestion from asynchronous processing. ClickHouse stores and aggregates usage data, while PostgreSQL remains the source of truth for transactional product and billing state. Dashed edges are optional, configuration-dependent flows: event deduplication is disabled by default and uses an in-process memory store unless the Redis driver is configured, and the API server can additionally use Redis for ingest-side deduplication and for tracking long-running query progress. The notification service evaluates rules and records pending notification events; actual webhook delivery to Svix runs as a leader-elected reconciler inside the API server. A separate optional Collector component (`collector/`) can buffer and forward events from external sources to the ingest API.
