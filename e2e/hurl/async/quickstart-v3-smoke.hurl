# Quickstart Smoke (v3) — ingest CloudEvents, wait for sink, query the meter
#
# Mirrors quickstart/quickstart_test.go (v3 portion only).
# Touches: meters, events, meter query.
#   POST /openmeter/meters          (v3, create)
#   POST /api/v1/events             (v1 ingest — only public ingest path)
#   POST /openmeter/meters/{id}/query (v3, query)
#
# This test exercises the full async ingest pipeline:
#   client → /api/v1/events → Kafka → sink-worker → ClickHouse → meter query.
#
# Local prereqs (in addition to `make up` + `make server`):
#   make sink-worker    # run in a separate terminal
# Without sink-worker the events never land in ClickHouse and the query
# retries will exhaust waiting for two data points to appear.
#
# Run:
#   hurl --test \
#     --variable base_url=http://localhost:8888 \
#     --variable api_key="" \
#     --variable run_id=$(date +%s%N | head -c 13) \
#     e2e/hurl/async/quickstart-v3-smoke.hurl

# ── 1. Create a v3 count meter with method+route dimensions ───────────────────
POST {{base_url}}/api/v3/openmeter/meters
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "key": "quickstart_v3_{{run_id}}",
  "name": "Quickstart V3 Meter {{run_id}}",
  "aggregation": "count",
  "event_type": "request",
  "dimensions": {
    "method": "$.method",
    "route":  "$.route"
  }
}
```
HTTP 201
[Captures]
meter_id: jsonpath "$.id"
[Asserts]
jsonpath "$.id" isString
jsonpath "$.aggregation" == "count"

# ── 2. Ingest event 1 — day 1 ─────────────────────────────────────────────────
# Retry on transient 5xx during sink-worker warmup; idempotent (event id is the
# dedup key, so re-POSTing the same id is a no-op).
POST {{base_url}}/api/v1/events
Authorization: Bearer {{api_key}}
Content-Type: application/cloudevents+json
[Options]
retry: 30
retry-interval: 1000
```json
{
  "specversion": "1.0",
  "id": "qs_{{run_id}}_00001",
  "source": "service-0",
  "type": "request",
  "subject": "customer-1",
  "time": "2023-01-01T00:00:00Z",
  "datacontenttype": "application/json",
  "data": {
    "method": "GET",
    "route": "/hello",
    "duration_ms": "40"
  }
}
```
HTTP 204

# ── 3. Ingest event 2 — day 1 (same hour as event 1) ─────────────────────────
POST {{base_url}}/api/v1/events
Authorization: Bearer {{api_key}}
Content-Type: application/cloudevents+json
[Options]
retry: 30
retry-interval: 1000
```json
{
  "specversion": "1.0",
  "id": "qs_{{run_id}}_00002",
  "source": "service-0",
  "type": "request",
  "subject": "customer-1",
  "time": "2023-01-01T00:00:00Z",
  "datacontenttype": "application/json",
  "data": {
    "method": "GET",
    "route": "/hello",
    "duration_ms": "40"
  }
}
```
HTTP 204

# ── 4. Ingest event 3 — day 2 ─────────────────────────────────────────────────
POST {{base_url}}/api/v1/events
Authorization: Bearer {{api_key}}
Content-Type: application/cloudevents+json
[Options]
retry: 30
retry-interval: 1000
```json
{
  "specversion": "1.0",
  "id": "qs_{{run_id}}_00003",
  "source": "service-0",
  "type": "request",
  "subject": "customer-1",
  "time": "2023-01-02T00:00:00Z",
  "datacontenttype": "application/json",
  "data": {
    "method": "GET",
    "route": "/hello",
    "duration_ms": "40"
  }
}
```
HTTP 204

# ── 5. Query meter at PT1H granularity — wait for sink to process the events ─
# Retries the entire request (status + asserts) until two windows appear with
# values 2 and 1 respectively. Up to 60s ceiling — matches the Go test's
# EventuallyWithT(30s, 1s) but doubled to give the sink-worker more headroom
# under load.
POST {{base_url}}/api/v3/openmeter/meters/{{meter_id}}/query
Authorization: Bearer {{api_key}}
Content-Type: application/json
[Options]
retry: 60
retry-interval: 1000
```json
{
  "granularity": "PT1H"
}
```
HTTP 200
[Asserts]
jsonpath "$.data" count == 2
jsonpath "$.data[0].value" == "2"
jsonpath "$.data[1].value" == "1"
