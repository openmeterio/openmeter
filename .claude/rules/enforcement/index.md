# Enforcement Rules — Index

This project has 162 rules across 12 project topics. Load only the topic file(s) relevant to your task. Universal Archie anti-patterns live in `universal.md` and apply to every project.

## By topic

| Topic | File | Rules |
|-------|------|-------|
| billing | by-topic/billing.md | 3 |
| concurrency | by-topic/concurrency.md | 2 |
| data-access | by-topic/data-access.md | 8 |
| data-modeling | by-topic/data-modeling.md | 6 |
| dependencies | by-topic/dependencies.md | 7 |
| layering | by-topic/layering.md | 11 |
| misc | by-topic/misc.md | 90 |
| schema-evolution | by-topic/schema-evolution.md | 1 |
| security | by-topic/security.md | 1 |
| services | by-topic/services.md | 1 |
| state-management | by-topic/state-management.md | 1 |
| testing | by-topic/testing.md | 1 |
| Universal | universal.md | 30 |

## By path

When editing a file matching one of these globs, load the listed topics first.

| Path glob | Topics to load |
|-----------|----------------|
| `**/*.go` | misc |
| `.github/workflows/ci.yaml` | misc |
| `api/openapi.cloud.yaml` | misc |
| `api/openapi.yaml` | misc |
| `api/spec/packages/aip/src/**/*.tsp` | misc |
| `api/spec/packages/legacy/src/**/*.tsp` | misc |
| `api/v3/handlers/**/*.go` | misc |
| `api/v3/openapi.yaml` | misc |
| `api/v3/server/**/*.go` | misc |
| `app/common/*.go` | misc |
| `app/common/customer.go` | misc |
| `cmd/**/*.go` | dependencies, misc |
| `cmd/balance-worker/main.go` | services |
| `cmd/billing-worker/**/*.go` | misc |
| `cmd/billing-worker/main.go` | misc, services |
| `cmd/server/**/*.go` | misc |
| `cmd/sink-worker/main.go` | services |
| `deploy/charts/benthos-collector/Chart.yaml` | misc |
| `deploy/charts/openmeter/Chart.yaml` | misc |
| `openmeter/**/*.go` | dependencies, layering, misc |
| `openmeter/**/adapter/**/*.go` | layering, misc |
| `openmeter/**/httpdriver/**/*.go` | misc |
| `openmeter/**/service.go` | layering |
| `openmeter/**/service/*.go` | data-access, misc |
| `openmeter/app/stripe/**/*.go` | concurrency |
| `openmeter/billing/**/*.go` | billing, misc |
| `openmeter/billing/**/*_test.go` | misc |
| `openmeter/billing/charges/**/*.go` | misc |
| `openmeter/billing/charges/**/*_test.go` | misc |
| `openmeter/billing/charges/**/adapter/**/*.go` | misc |
| `openmeter/billing/validators/**/*.go` | misc |
| `openmeter/billing/worker/**/*.go` | misc |
| `openmeter/billing/worker/advance/**/*.go` | misc |
| `openmeter/credit/**/*.go` | billing |
| `openmeter/dedupe/**/*.go` | data-access |
| `openmeter/ent/schema/*.go` | data-access, misc |
| `openmeter/ent/schema/balance_snapshot.go` | data-modeling |
| `openmeter/ent/schema/billing.go` | schema-evolution |
| `openmeter/ent/schema/chargemeta.go` | data-modeling |
| `openmeter/ent/schema/charges*.go` | data-modeling |
| `openmeter/ent/schema/charges.go` | data-modeling |
| `openmeter/ent/schema/ledger_account.go` | data-modeling |
| `openmeter/ent/schema/ledger_customer_account.go` | data-modeling |
| `openmeter/ent/schema/notification.go` | data-modeling |
| `openmeter/ent/schema/subscription.go` | data-modeling |
| `openmeter/entitlement/**/*.go` | billing |
| `openmeter/entitlement/balanceworker/**/*.go` | state-management |
| `openmeter/ingest/**/*.go` | data-access |
| `openmeter/ledger/**/*.go` | misc |
| `openmeter/notification/**/*.go` | misc |
| `openmeter/portal/**/*.go` | security |
| `openmeter/productcatalog/feature/**/*.go` | data-modeling |
| `openmeter/server/**/*.go` | concurrency, misc |
| `openmeter/sink/**/*.go` | data-access, misc |
| `openmeter/streaming/**/*.go` | data-access |
| `openmeter/streaming/clickhouse/**/*.go` | data-access |
| `openmeter/subscription/**/*.go` | misc |
| `openmeter/watermill/**/*.go` | misc |
| `openmeter/watermill/eventbus/**/*.go` | misc |
