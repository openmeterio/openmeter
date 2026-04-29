# Features E2E — feature lifecycle (p0)
#
# Covers: feature_lifecycle (p0) — see e2e/specs/features.md
#   POST   /openmeter/features
#   GET    /openmeter/features/{id}
#   GET    /openmeter/features
#   PATCH  /openmeter/features/{id}     (set unit_cost)
#   PATCH  /openmeter/features/{id}     (clear unit_cost = null)
#   DELETE /openmeter/features/{id}
#   GET    /openmeter/features/{id}     (404 after delete — features are hard-deleted)
#
# Run:
#   hurl --test \
#     --variable base_url=http://localhost:8888 \
#     --variable api_key="" \
#     --variable run_id=$(date +%s%N | head -c 13) \
#     e2e/hurl/openmeter-v3-features.hurl

# ── 1. Create static feature ──────────────────────────────────────────────────
# Baseline: no meter, no unit_cost. Key must NOT parse as a ULID — prefix with
# "feat_" to guarantee that.
POST {{base_url}}/api/v3/openmeter/features
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "key": "feat_{{run_id}}",
  "name": "Test Feature"
}
```
HTTP 201
[Captures]
feature_id: jsonpath "$.id"
feature_key: jsonpath "$.key"
[Asserts]
jsonpath "$.id" isString
jsonpath "$.key" == "feat_{{run_id}}"
jsonpath "$.name" == "Test Feature"
jsonpath "$.meter" not exists
jsonpath "$.unit_cost" not exists
jsonpath "$.deleted_at" not exists

# ── 2. Get feature by id ──────────────────────────────────────────────────────
GET {{base_url}}/api/v3/openmeter/features/{{feature_id}}
Authorization: Bearer {{api_key}}
HTTP 200
[Asserts]
jsonpath "$.id" == "{{feature_id}}"
jsonpath "$.key" == "feat_{{run_id}}"
jsonpath "$.name" == "Test Feature"

# ── 3. List features — created feature appears ───────────────────────────────
# Page size 1000 because the shared DB may push fresh rows past page-1 default.
GET {{base_url}}/api/v3/openmeter/features
Authorization: Bearer {{api_key}}
[QueryStringParams]
page[size]: 1000
HTTP 200
[Asserts]
jsonpath "$.data" isCollection
jsonpath "$.data[?(@.id=='{{feature_id}}')].id" contains "{{feature_id}}"

# ── 4. Patch — set manual unit_cost ──────────────────────────────────────────
# UpdateFeatureRequest.unit_cost uses tri-state semantics: omitting the key is
# rejected (covered by feature_update_unit_cost_required scenario). Here we set
# a manual cost.
PATCH {{base_url}}/api/v3/openmeter/features/{{feature_id}}
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "unit_cost": {
    "type": "manual",
    "amount": "5"
  }
}
```
HTTP 200
[Asserts]
jsonpath "$.id" == "{{feature_id}}"
jsonpath "$.unit_cost.type" == "manual"
jsonpath "$.unit_cost.amount" == "5"

# ── 5. Get — verify unit_cost persisted ──────────────────────────────────────
GET {{base_url}}/api/v3/openmeter/features/{{feature_id}}
Authorization: Bearer {{api_key}}
HTTP 200
[Asserts]
jsonpath "$.unit_cost.type" == "manual"
jsonpath "$.unit_cost.amount" == "5"

# ── 6. Patch — clear unit_cost (null) ────────────────────────────────────────
# Passing null clears the unit_cost. Response should have unit_cost absent.
PATCH {{base_url}}/api/v3/openmeter/features/{{feature_id}}
Authorization: Bearer {{api_key}}
Content-Type: application/json
```json
{
  "unit_cost": null
}
```
HTTP 200
[Asserts]
jsonpath "$.id" == "{{feature_id}}"
jsonpath "$.unit_cost" not exists

# ── 7. Get — verify unit_cost cleared ────────────────────────────────────────
GET {{base_url}}/api/v3/openmeter/features/{{feature_id}}
Authorization: Bearer {{api_key}}
HTTP 200
[Asserts]
jsonpath "$.unit_cost" not exists

# ── 8. Delete feature ────────────────────────────────────────────────────────
DELETE {{base_url}}/api/v3/openmeter/features/{{feature_id}}
Authorization: Bearer {{api_key}}
HTTP 204

# ── 9. Get after delete — 404 (features are hard-deleted) ────────────────────
# Contrast with meters/plans, which soft-delete and return 200 + deleted_at.
GET {{base_url}}/api/v3/openmeter/features/{{feature_id}}
Authorization: Bearer {{api_key}}
HTTP 404
[Asserts]
jsonpath "$.detail" contains "feature not found: {{feature_id}}"
