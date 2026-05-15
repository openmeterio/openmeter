# Enforcement Rules — Index

This project has 142 rules across 8 project topics. Load only the topic file(s) relevant to your task. Universal Archie anti-patterns live in `universal.md` and apply to every project.

## By topic

| Topic | File | Rules |
|-------|------|-------|
| billing | by-topic/billing.md | 2 |
| concurrency | by-topic/concurrency.md | 1 |
| data-access | by-topic/data-access.md | 3 |
| dependencies | by-topic/dependencies.md | 7 |
| layering | by-topic/layering.md | 7 |
| misc | by-topic/misc.md | 90 |
| security | by-topic/security.md | 1 |
| state-management | by-topic/state-management.md | 1 |
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
| `cmd/billing-worker/**/*.go` | misc |
| `cmd/billing-worker/main.go` | misc |
| `cmd/server/**/*.go` | misc |
| `deploy/charts/benthos-collector/Chart.yaml` | misc |
| `deploy/charts/openmeter/Chart.yaml` | misc |
| `openmeter/**/*.go` | dependencies, misc |
| `openmeter/**/adapter/**/*.go` | layering, misc |
| `openmeter/**/httpdriver/**/*.go` | misc |
| `openmeter/**/service/*.go` | misc |
| `openmeter/billing/**/*.go` | billing, misc |
| `openmeter/billing/**/*_test.go` | misc |
| `openmeter/billing/charges/**/*.go` | misc |
| `openmeter/billing/charges/**/*_test.go` | misc |
| `openmeter/billing/charges/**/adapter/**/*.go` | misc |
| `openmeter/billing/validators/**/*.go` | misc |
| `openmeter/billing/worker/**/*.go` | misc |
| `openmeter/billing/worker/advance/**/*.go` | misc |
| `openmeter/credit/**/*.go` | billing |
| `openmeter/ent/schema/*.go` | misc |
| `openmeter/entitlement/**/*.go` | billing |
| `openmeter/entitlement/balanceworker/**/*.go` | state-management |
| `openmeter/ingest/**/*.go` | data-access |
| `openmeter/ledger/**/*.go` | misc |
| `openmeter/notification/**/*.go` | misc |
| `openmeter/portal/**/*.go` | security |
| `openmeter/server/**/*.go` | misc |
| `openmeter/sink/**/*.go` | data-access, misc |
| `openmeter/subscription/**/*.go` | misc |
| `openmeter/watermill/**/*.go` | misc |
| `openmeter/watermill/eventbus/**/*.go` | misc |
