# Enforcement Rules — Index

This project has 102 rules across 14 project topics. Load only the topic file(s) relevant to your task. Universal Archie anti-patterns live in `universal.md` and apply to every project.

## By topic

| Topic | File | Rules |
|-------|------|-------|
| billing-lifecycle | by-topic/billing-lifecycle.md | 3 |
| codegen | by-topic/codegen.md | 4 |
| concurrency | by-topic/concurrency.md | 4 |
| data-access | by-topic/data-access.md | 4 |
| data-modeling | by-topic/data-modeling.md | 8 |
| dependencies | by-topic/dependencies.md | 6 |
| error-handling | by-topic/error-handling.md | 3 |
| layering | by-topic/layering.md | 11 |
| mapping | by-topic/mapping.md | 2 |
| messaging | by-topic/messaging.md | 3 |
| schema-evolution | by-topic/schema-evolution.md | 5 |
| security | by-topic/security.md | 1 |
| services | by-topic/services.md | 12 |
| testing | by-topic/testing.md | 6 |
| Universal | universal.md | 30 |

## By path

When editing a file matching one of these globs, load the listed topics first.

| Path glob | Topics to load |
|-----------|----------------|
| `**/*.gen.go` | codegen |
| `**/wire_gen.go` | codegen |
| `api/**` | concurrency, error-handling |
| `api/api.gen.go` | codegen |
| `api/client/go/client.gen.go` | codegen |
| `api/v3/api.gen.go` | codegen |
| `api/v3/handlers/**` | services |
| `app/**` | concurrency, error-handling, services |
| `cmd/**` | error-handling |
| `openmeter/**` | concurrency, error-handling, services |
| `openmeter/**/adapter/**` | data-access |
| `openmeter/**/service/**` | services |
| `openmeter/billing/charges/**/adapter/**` | data-access |
| `openmeter/ent/db/**` | codegen |
| `pkg/**` | concurrency, error-handling, services |
