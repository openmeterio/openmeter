# seeder

<!-- archie:ai-start -->

> Benthos/Redpanda Connect pipeline config that generates synthetic CloudEvents and POSTs them to the OpenMeter ingest API. Used exclusively for local quickstart demos — not production code.

## Patterns

**Environment-variable-driven configuration** — Every runtime-variable value uses Benthos interpolation syntax ${VAR:default}; new tunables must follow this — no hardcoded values. (`count: ${SEEDER_COUNT:0}
interval: "${SEEDER_INTERVAL:50ms}"`)
**CloudEvents 1.0 envelope** — The generate mapping must produce a root object with id, specversion, type, source, subject, time, and data fields to be accepted by /api/v1/events. (`root = {"id": uuid_v4(), "specversion": "1.0", "type": $event_type, "source": $source, "subject": $subject, "time": $time, "data": {...}}`)
**Switch output for optional stdout logging** — The output.switch pattern with a SEEDER_LOG guard is the canonical way to add optional debug output; new output targets must be added as an additional case, not a replacement. (`- check: '"${SEEDER_LOG:false}" == "true"'
  output:
    stdout:
      codec: lines`)
**Bearer token auth on HTTP output** — The Authorization header must use Bearer scheme with the OPENMETER_TOKEN env var; never embed tokens directly. (`Authorization: "Bearer ${OPENMETER_TOKEN:}"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `config.yaml` | Single Benthos pipeline: generate input → CloudEvent Bloblang mapping → HTTP POST output with optional stdout logging. | SEEDER_COUNT=0 means generate forever (set non-zero to cap). The time field is backdated up to 3 days via ts_sub_iso8601("P3D") to spread events — changing this affects meter aggregation windows in demos. |

## Anti-Patterns

- Hardcoding OPENMETER_URL or OPENMETER_TOKEN instead of using env-var interpolation.
- Removing the specversion or id fields from the generated CloudEvent — the ingest API rejects the event.
- Adding business logic or data transformations beyond synthetic seeding — this config is demo-only, not a reusable template.
- Setting max_in_flight > 1 on the http_client output without understanding back-pressure against a local server.

## Decisions

- **Use Benthos generate input with Bloblang mapping rather than a separate Go seed program** — Keeps the quickstart self-contained with no extra binary; Benthos is already a project dependency (benthos-collector), so no new toolchain is introduced.
- **Backdate event time by up to 3 days with random spread** — Produces realistic historical usage data in meter aggregation windows on first run, avoiding an empty dashboard on quickstart.

## Example: CloudEvent mapping with backdated time and env-var-driven subject count

```
mapping: |
  let max_subjects = ${SEEDER_MAX_SUBJECTS:10}
  let subject = "customer-%d".format(random_int(seed: timestamp_unix_nano()) % $max_subjects)
  let time = (now().ts_sub_iso8601("P3D").ts_unix() + random_int(min: 60, max: 60 * 60 * 24 * 3)).ts_format()
  root = {
    "id": uuid_v4(),
    "specversion": "1.0",
    "type": "request",
    "source": "api-gateway",
    "subject": $subject,
    "time": $time,
    "data": {"method": "GET", "path": "/", "region": "us-east-1"}
  }
```

<!-- archie:ai-end -->
