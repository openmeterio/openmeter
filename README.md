<div align="center">

![OpenMeter logo](assets/logo.png)

# OpenMeter

The open-source metering and billing platform
for AI, agentic and DevTool monetization.

[Docs](https://openmeter.io/docs) |
[Hosted](https://cloud.konghq.com/register?utm_campaign=metering_and_billing) |
[Blog](https://openmeter.io/blog) |
[Contributing](CONTRIBUTING.md)

[![GitHub Release](https://img.shields.io/github/v/release/openmeterio/openmeter?style=flat-square)](https://github.com/openmeterio/openmeter/releases/latest)
[![CI Status](https://img.shields.io/github/actions/workflow/status/openmeterio/openmeter/ci.yaml?style=flat-square)](https://github.com/openmeterio/openmeter/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/openmeterio/openmeter?style=flat-square)](https://goreportcard.com/report/github.com/openmeterio/openmeter)
![GitHub Stars](https://img.shields.io/github/stars/openmeterio/openmeter?style=flat-square)

</div>

---

OpenMeter is a real-time metering and billing engine that
helps you track usage, enforce limits, manage subscriptions,
and automate invoicing — all in one platform. Ingest events
via a simple API, define meters with flexible aggregations,
and connect usage data to billing, entitlements, and
customer-facing dashboards.

## Features

- **Usage Metering** — Ingest events in
  [CloudEvents](https://cloudevents.io) format, define meters
  with flexible aggregations (SUM, COUNT, AVG, MIN, MAX),
  and query usage in real time.
- **Usage-Based Billing** — Generate invoices from metered
  usage. Supports tiered, graduated, and flat-fee pricing
  with automated invoice lifecycle management.
- **Usage Limits and Entitlements** — Enforce usage quotas
  per feature with real-time balance tracking, boolean
  feature flags, and grace periods.
- **Product Catalog** — Define plans, add-ons, features, and
  rate cards. Manage subscriptions with mid-cycle changes,
  prorating, and alignment.
- **Prepaid Credits** — Support paid or promotional credit grants
  with priority-based burn-down and expiration.
- **Customer Portal** — Token-based self-service dashboards
  so your customers can see their own usage.
- **Notifications** — Webhook-based alerts with configurable
  rules and channels for usage thresholds and billing events.
- **LLM Cost Tracking** — First-class support for metering
  AI token usage and computing model-specific costs.

## Getting Started

### Cloud

The fastest way to start.
[Start for free](https://cloud.konghq.com/register?utm_campaign=metering_and_billing)
and begin metering and billing in minutes —
no infrastructure to manage.

### Self-Hosted

Run OpenMeter locally with Docker Compose:

```sh
git clone git@github.com:openmeterio/openmeter.git
cd openmeter/quickstart
docker compose up -d
```

Then ingest your first event:

```sh
curl -X POST http://localhost:48888/api/v1/events \
  -H 'Content-Type: application/cloudevents+json' \
  --data-raw '{
    "specversion": "1.0",
    "type": "request",
    "id": "00001",
    "time": "2024-01-01T00:00:00.001Z",
    "source": "my-service",
    "subject": "customer-1",
    "data": { "method": "GET", "route": "/api/hello" }
  }'
```

Query your usage:

```sh
curl 'http://localhost:48888/api/v1/meters/api_requests_total/query?windowSize=HOUR' | jq
```

See the full [quickstart guide](/quickstart) for more details.

### Deploy to Production

Deploy to Kubernetes using our
[Helm chart](https://openmeter.io/docs/deploy/kubernetes).

## SDKs

| Language             | Package                                                                        | Source                                             |
|----------------------|--------------------------------------------------------------------------------|----------------------------------------------------|
| Go                   | [openmeter](https://pkg.go.dev/github.com/openmeterio/openmeter/api/client/go) | [api/client/go](/api/client/go)                    |
| JavaScript / Node.js | [@openmeter/sdk](https://www.npmjs.com/package/@openmeter/sdk)                 | [api/client/javascript](/api/client/javascript)    |
| Python               | [openmeter](https://pypi.org/project/openmeter)                                | [api/client/python](/api/client/python)            |

Don't see your language? Use the
[OpenAPI spec](https://github.com/openmeterio/openmeter/blob/main/api/openapi.yaml)
directly or
[request an SDK](https://github.com/openmeterio/openmeter/issues/new?assignees=&labels=area%2Fapi%2Ckind%2Ffeature&projects=&template=feature_request.yaml).

## Architecture

OpenMeter is built in Go with a stack optimized for
high-volume event ingestion and real-time aggregation:

| Component                | Role                                                     |
|--------------------------|----------------------------------------------------------|
| **PostgreSQL** (Ent ORM) | Billing, subscriptions, entitlements, product catalog    |
| **ClickHouse**           | Real-time usage aggregation and analytics                |
| **Kafka**                | Event streaming and ingestion pipeline                   |
| **TypeSpec**             | API-first design — OpenAPI spec and SDKs from TypeSpec   |

## Community

We'd love to have you involved:

- **[Contributing](CONTRIBUTING.md)** — Start here if you
  want to contribute code.
- **[Code of Conduct](CODE_OF_CONDUCT.md)** — Our community
  guidelines.
- **[Blog](https://openmeter.io/blog)** — Product updates
  and engineering deep dives.

## Development

Prerequisites: [Nix](https://nixos.org/download.html) and
[direnv](https://direnv.net/docs/installation.html) are
recommended. See [CONTRIBUTING.md](CONTRIBUTING.md) for
detailed setup instructions. The Nix shell provides `task`;
install [Task](https://taskfile.dev/docs/installation) separately
if you do not use Nix.

```sh
task up       # Start dependencies (Postgres, Kafka, ClickHouse)
task server   # Run the API server with hot reload
task test     # Run tests
task lint     # Run linters
```

## License

Licensed under [Apache 2.0](LICENSE).

[![FOSSA Status](https://app.fossa.com/api/projects/custom%2B38090%2Fgithub.com%2Fopenmeterio%2Fopenmeter.svg?type=large)](https://app.fossa.com/projects/custom%2B38090%2Fgithub.com%2Fopenmeterio%2Fopenmeter?ref=badge_large)
