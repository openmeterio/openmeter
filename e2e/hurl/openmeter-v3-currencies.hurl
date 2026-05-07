# Currencies E2E — custom currency lifecycle + cost bases
#
# Covers: currency_custom_lifecycle (p0)
#   POST /openmeter/currencies/custom
#   GET  /openmeter/currencies
#   POST /openmeter/currencies/custom/{id}/cost-bases
#   GET  /openmeter/currencies/custom/{id}/cost-bases
#
# Run:
#   hurl --test \
#     --variable base_url=http://localhost:8888 \
#     --variable run_id=$(date +%s%N | head -c 13) \
#     e2e/hurl/openmeter-v3-currencies.hurl

# ── 1. Create custom currency ─────────────────────────────────────────────────
POST {{base_url}}/api/v3/openmeter/currencies/custom
Content-Type: application/json
```json
{
  "code": "TC{{run_id}}",
  "name": "Test Currency {{run_id}}",
  "symbol": "T$"
}
```
HTTP 201
[Captures]
currency_id: jsonpath "$.id"
currency_code: jsonpath "$.code"
[Asserts]
jsonpath "$.id" isString
jsonpath "$.type" == "custom"
jsonpath "$.code" == "TC{{run_id}}"
jsonpath "$.name" == "Test Currency {{run_id}}"
jsonpath "$.symbol" == "T$"

# ── 2. List all currencies — custom currency present ─────────────────────────
# Use JSONPath filter expression — avoids $.data[*].id type inconsistency
# (returns string for 1-item arrays, breaking contains predicate)
GET {{base_url}}/api/v3/openmeter/currencies
[QueryStringParams]
page[size]: 1000
HTTP 200
[Asserts]
jsonpath "$.data" isCollection
jsonpath "$.data[?(@.id=='{{currency_id}}')].id" contains "{{currency_id}}"

# ── 3. List filtered by type=custom — currency present, all entries custom ───
GET {{base_url}}/api/v3/openmeter/currencies
[QueryStringParams]
filter[type]: custom
page[size]: 1000
HTTP 200
[Asserts]
jsonpath "$.data[?(@.id=='{{currency_id}}')].id" contains "{{currency_id}}"
jsonpath "$.data[0].type" == "custom"

# ── 4. List filtered by type=fiat — only fiat currencies returned ─────────────
# Fiat currencies have no id field; assert type of first result is fiat
GET {{base_url}}/api/v3/openmeter/currencies
[QueryStringParams]
filter[type]: fiat
page[size]: 10
HTTP 200
[Asserts]
jsonpath "$.data" isCollection
jsonpath "$.data[0].type" == "fiat"

# ── 5. Create cost basis ──────────────────────────────────────────────────────
POST {{base_url}}/api/v3/openmeter/currencies/custom/{{currency_id}}/cost-bases
Content-Type: application/json
```json
{
  "fiat_code": "USD",
  "rate": "1.5"
}
```
HTTP 201
[Captures]
cost_basis_id: jsonpath "$.id"
[Asserts]
jsonpath "$.id" isString
jsonpath "$.fiat_code" == "USD"
jsonpath "$.rate" == "1.5"

# ── 6. List cost bases — cost basis present ───────────────────────────────────
GET {{base_url}}/api/v3/openmeter/currencies/custom/{{currency_id}}/cost-bases
HTTP 200
[Asserts]
jsonpath "$.data" isCollection
jsonpath "$.data[?(@.id=='{{cost_basis_id}}')].id" contains "{{cost_basis_id}}"
jsonpath "$.data[0].fiat_code" == "USD"
jsonpath "$.data[0].rate" == "1.5"

# ── 7. Duplicate code rejected with 409 ──────────────────────────────────────
POST {{base_url}}/api/v3/openmeter/currencies/custom
Content-Type: application/json
```json
{
  "code": "TC{{run_id}}",
  "name": "Duplicate Currency",
  "symbol": "D$"
}
```
HTTP 409
[Asserts]
jsonpath "$.detail" contains "already exists"
