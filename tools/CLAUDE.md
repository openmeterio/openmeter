# tools

<!-- archie:ai-start -->

> Organisational folder owning two sub-concerns: database migration tooling (tools/migrate) and a bash readiness helper (wait-for-compose.sh). Acts as the operational boundary between schema authorship (Ent/Atlas) and runtime startup; nothing in tools/ contains business logic.

## Patterns

**wait-for-compose health check** — wait-for-compose.sh polls docker compose container health status (healthy/running) before CI steps proceed. Uses `docker compose ps -q` + `docker inspect` in a 60-attempt loop with 2s sleep. (`./tools/wait-for-compose.sh postgres svix redis`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `tools/wait-for-compose.sh` | CI readiness gate: blocks until listed docker compose services are healthy or running. Used by Makefile targets before running tests. | Containers with no healthcheck fall through to state-based check (running/restarting/created). Exit 1 on unhealthy or timeout after 120s. |

## Anti-Patterns

- Adding business logic or domain imports to anything under tools/
- Hand-editing tools/migrate/migrations/ — Atlas owns the chain and atlas.sum will fail CI
- Calling atlas migrate diff expecting new ent.View schemas to appear — views require viewgen
- Using ORM (Ent/GORM) inside migration stop tests — raw *sql.DB is required

## Decisions

- **wait-for-compose.sh is a standalone bash script rather than a Go helper** — It must run before any Go binary is available; bash docker-inspect polling has no runtime dependencies.

<!-- archie:ai-end -->
