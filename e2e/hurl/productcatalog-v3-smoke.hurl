# Product Catalog Smoke (v3) — cross-cutting plan + addon authoring flow
#
# Mirrors e2e/productcatalog_smoke_v3_test.go (TestV3ProductCatalogSmoke).
# Touches: meters, features, plans, plan-addons, addons.
#   POST /openmeter/meters
#   POST /openmeter/features
#   POST /openmeter/plans
#   PUT  /openmeter/plans/{id}
#   GET  /openmeter/plans/{id}
#   POST /openmeter/plans/{id}/publish
#   POST /openmeter/addons
#   POST /openmeter/addons/{id}/publish
#   POST /openmeter/plans/{id}/addons
#   GET  /openmeter/plans/{id}/addons
#
# Decimal note: prices are pre-normalized ("0.1", not "0.10") so re-sending the
# same body across PUTs stays stable — the server canonicalizes decimals on
# round-trip and the Go test pulls rate cards back from GET to avoid drift.
#
# Run:
#   hurl --test \
#     --variable base_url=http://localhost:8888 \
#     --variable api_key="" \
#     --variable run_id=$(date +%s%N | head -c 13) \
#     e2e/hurl/productcatalog-v3-smoke.hurl

# ── 1. Create a meter ─────────────────────────────────────────────────────────
POST {{base_url}}/api/v3/openmeter/meters
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "key": "pc_smoke_meter_{{run_id}}",
  "name": "PC Smoke Meter {{run_id}}",
  "aggregation": "sum",
  "event_type": "pc_smoke_event_{{run_id}}",
  "value_property": "$.value"
}
```
HTTP 201
[Captures]
meter_id: jsonpath "$.id"
[Asserts]
jsonpath "$.id" isString
jsonpath "$.key" == "pc_smoke_meter_{{run_id}}"

# ── 2. Create a feature bound to the meter ────────────────────────────────────
POST {{base_url}}/api/v3/openmeter/features
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "key": "pc_smoke_feature_{{run_id}}",
  "name": "PC Smoke Feature {{run_id}}",
  "meter": {
    "id": "{{meter_id}}"
  }
}
```
HTTP 201
[Captures]
feature_id: jsonpath "$.id"
[Asserts]
jsonpath "$.id" isString
jsonpath "$.key" == "pc_smoke_feature_{{run_id}}"
jsonpath "$.meter.id" == "{{meter_id}}"

# ── 3. Create a draft plan with a single flat rate card ───────────────────────
POST {{base_url}}/api/v3/openmeter/plans
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "key": "pc_smoke_plan_{{run_id}}",
  "name": "PC Smoke Plan {{run_id}}",
  "currency": "USD",
  "billing_cadence": "P1M",
  "phases": [
    {
      "key": "pc_smoke_phase_{{run_id}}",
      "name": "PC Smoke Phase",
      "rate_cards": [
        {
          "key": "flat_{{run_id}}",
          "name": "Flat Fee",
          "price": { "type": "flat", "amount": "10" },
          "billing_cadence": "P1M",
          "payment_term": "in_advance"
        }
      ]
    }
  ]
}
```
HTTP 201
[Captures]
plan_id: jsonpath "$.id"
[Asserts]
jsonpath "$.id" isString
jsonpath "$.status" == "draft"
jsonpath "$.phases" count == 1
jsonpath "$.phases[0].rate_cards" count == 1
jsonpath "$.phases[0].key" == "pc_smoke_phase_{{run_id}}"

# ── 4. Update the plan: flat + unit-with-feature + graduated rate cards ───────
PUT {{base_url}}/api/v3/openmeter/plans/{{plan_id}}
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "name": "PC Smoke Plan {{run_id}}",
  "phases": [
    {
      "key": "pc_smoke_phase_{{run_id}}",
      "name": "PC Smoke Phase",
      "rate_cards": [
        {
          "key": "flat_{{run_id}}",
          "name": "Flat Fee",
          "price": { "type": "flat", "amount": "10" },
          "billing_cadence": "P1M",
          "payment_term": "in_advance"
        },
        {
          "key": "usage_{{run_id}}",
          "name": "Usage Unit Fee",
          "feature": { "id": "{{feature_id}}" },
          "price": { "type": "unit", "amount": "0.1" },
          "billing_cadence": "P1M",
          "payment_term": "in_arrears"
        },
        {
          "key": "graduated_{{run_id}}",
          "name": "Graduated Fee",
          "price": {
            "type": "graduated",
            "tiers": [
              { "up_to_amount": "100", "unit_price": { "type": "unit", "amount": "0.1" } },
              { "unit_price": { "type": "unit", "amount": "0.05" } }
            ]
          },
          "billing_cadence": "P1M",
          "payment_term": "in_arrears"
        }
      ]
    }
  ]
}
```
HTTP 200
[Asserts]
jsonpath "$.phases" count == 1
jsonpath "$.phases[0].rate_cards" count == 3
jsonpath "$.phases[0].rate_cards[?(@.key=='usage_{{run_id}}')].feature.id" contains "{{feature_id}}"

# ── 5. Update the plan: append a defective P2W rate card (cadence misaligned)
# Plan cadence is P1M, so a P2W rate card surfaces rate_card_billing_cadence_unaligned.
# The PUT itself is accepted (200) — the defect appears as a draft validation_error.
PUT {{base_url}}/api/v3/openmeter/plans/{{plan_id}}
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "name": "PC Smoke Plan {{run_id}}",
  "phases": [
    {
      "key": "pc_smoke_phase_{{run_id}}",
      "name": "PC Smoke Phase",
      "rate_cards": [
        {
          "key": "flat_{{run_id}}",
          "name": "Flat Fee",
          "price": { "type": "flat", "amount": "10" },
          "billing_cadence": "P1M",
          "payment_term": "in_advance"
        },
        {
          "key": "usage_{{run_id}}",
          "name": "Usage Unit Fee",
          "feature": { "id": "{{feature_id}}" },
          "price": { "type": "unit", "amount": "0.1" },
          "billing_cadence": "P1M",
          "payment_term": "in_arrears"
        },
        {
          "key": "graduated_{{run_id}}",
          "name": "Graduated Fee",
          "price": {
            "type": "graduated",
            "tiers": [
              { "up_to_amount": "100", "unit_price": { "type": "unit", "amount": "0.1" } },
              { "unit_price": { "type": "unit", "amount": "0.05" } }
            ]
          },
          "billing_cadence": "P1M",
          "payment_term": "in_arrears"
        },
        {
          "key": "defective_{{run_id}}",
          "name": "Defective Cadence Fee",
          "price": { "type": "flat", "amount": "5" },
          "billing_cadence": "P2W",
          "payment_term": "in_advance"
        }
      ]
    }
  ]
}
```
HTTP 200
[Asserts]
jsonpath "$.phases[0].rate_cards" count == 4

# ── 6. GET plan — validation_errors surfaces the cadence-unaligned code ──────
GET {{base_url}}/api/v3/openmeter/plans/{{plan_id}}
Authorization: Bearer {{api_key}}
HTTP 200
[Asserts]
jsonpath "$.status" == "draft"
jsonpath "$.validation_errors[?(@.code=='rate_card_billing_cadence_unaligned')].code" contains "rate_card_billing_cadence_unaligned"

# ── 7. Publish — rejected with the same domain code ──────────────────────────
POST {{base_url}}/api/v3/openmeter/plans/{{plan_id}}/publish
Authorization: Bearer {{api_key}}
HTTP 400
[Asserts]
jsonpath "$.extensions.validationErrors[?(@.code=='rate_card_billing_cadence_unaligned')].code" contains "rate_card_billing_cadence_unaligned"

# ── 8. Update the plan: remove the defective rate card ───────────────────────
PUT {{base_url}}/api/v3/openmeter/plans/{{plan_id}}
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "name": "PC Smoke Plan {{run_id}}",
  "phases": [
    {
      "key": "pc_smoke_phase_{{run_id}}",
      "name": "PC Smoke Phase",
      "rate_cards": [
        {
          "key": "flat_{{run_id}}",
          "name": "Flat Fee",
          "price": { "type": "flat", "amount": "10" },
          "billing_cadence": "P1M",
          "payment_term": "in_advance"
        },
        {
          "key": "usage_{{run_id}}",
          "name": "Usage Unit Fee",
          "feature": { "id": "{{feature_id}}" },
          "price": { "type": "unit", "amount": "0.1" },
          "billing_cadence": "P1M",
          "payment_term": "in_arrears"
        },
        {
          "key": "graduated_{{run_id}}",
          "name": "Graduated Fee",
          "price": {
            "type": "graduated",
            "tiers": [
              { "up_to_amount": "100", "unit_price": { "type": "unit", "amount": "0.1" } },
              { "unit_price": { "type": "unit", "amount": "0.05" } }
            ]
          },
          "billing_cadence": "P1M",
          "payment_term": "in_arrears"
        }
      ]
    }
  ]
}
```
HTTP 200
[Asserts]
jsonpath "$.phases[0].rate_cards" count == 3

# ── 9. GET plan — validation_errors no longer carries the cadence code ───────
# Robust against the field being absent OR an empty array OR populated by
# unrelated codes — only the one we just fixed is asserted absent.
GET {{base_url}}/api/v3/openmeter/plans/{{plan_id}}
Authorization: Bearer {{api_key}}
HTTP 200
[Asserts]
jsonpath "$.validation_errors[?(@.code=='rate_card_billing_cadence_unaligned')]" not exists

# ── 10. Create a draft addon ─────────────────────────────────────────────────
POST {{base_url}}/api/v3/openmeter/addons
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "key": "pc_smoke_addon_{{run_id}}",
  "name": "PC Smoke Addon {{run_id}}",
  "currency": "USD",
  "instance_type": "single",
  "rate_cards": [
    {
      "key": "addon_fee_{{run_id}}",
      "name": "Addon Flat Fee",
      "price": { "type": "flat", "amount": "10" },
      "billing_cadence": "P1M",
      "payment_term": "in_advance"
    }
  ]
}
```
HTTP 201
[Captures]
addon_id: jsonpath "$.id"
[Asserts]
jsonpath "$.id" isString
jsonpath "$.status" == "draft"

# ── 11. Publish the addon ────────────────────────────────────────────────────
POST {{base_url}}/api/v3/openmeter/addons/{{addon_id}}/publish
Authorization: Bearer {{api_key}}
HTTP 200
[Asserts]
jsonpath "$.id" == "{{addon_id}}"
jsonpath "$.status" == "active"

# ── 12. Attach the published addon to the still-draft plan ───────────────────
POST {{base_url}}/api/v3/openmeter/plans/{{plan_id}}/addons
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "name": "PC Smoke Plan Addon",
  "addon": { "id": "{{addon_id}}" },
  "from_plan_phase": "pc_smoke_phase_{{run_id}}"
}
```
HTTP 201
[Captures]
plan_addon_id: jsonpath "$.id"
[Asserts]
jsonpath "$.id" isString
jsonpath "$.addon.id" == "{{addon_id}}"
jsonpath "$.from_plan_phase" == "pc_smoke_phase_{{run_id}}"

# ── 13. Publish the plan — addon attached, plan goes active ──────────────────
POST {{base_url}}/api/v3/openmeter/plans/{{plan_id}}/publish
Authorization: Bearer {{api_key}}
HTTP 200
[Asserts]
jsonpath "$.id" == "{{plan_id}}"
jsonpath "$.status" == "active"
jsonpath "$.effective_from" isString

# ── 14. List plan-addons — attached addon survives publish ───────────────────
GET {{base_url}}/api/v3/openmeter/plans/{{plan_id}}/addons
Authorization: Bearer {{api_key}}
[QueryStringParams]
page[size]: 1000
HTTP 200
[Asserts]
jsonpath "$.data" isCollection
jsonpath "$.data[?(@.id=='{{plan_addon_id}}')].id" contains "{{plan_addon_id}}"
