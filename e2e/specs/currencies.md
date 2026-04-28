# E2E Scenario Specifications — Currencies

Natural-language, runner-agnostic description of e2e scenarios for the
`currencies` endpoint family. Each `## Scenario` describes wire-level
behavior (HTTP verb, path, status code, response shape,
`problem+json` error shape) that any downstream runner can translate
to an executable test.

See the `e2e-nl` skill (`.agents/skills/e2e-nl/`) for format rules
(`references/format.md`) and worked examples (`references/examples.md`).

**Endpoints covered:**
- `GET /openmeter/currencies` — list all currencies (fiat + custom)
- `POST /openmeter/currencies/custom` — create a custom currency
- `POST /openmeter/currencies/custom/{currencyId}/cost-bases` — create a cost basis for a custom currency
- `GET /openmeter/currencies/custom/{currencyId}/cost-bases` — list cost bases for a custom currency

**Custom vs fiat currencies:** Custom currencies have an `id` and `type: "custom"` in responses.
Fiat currencies (ISO-4217) are built-in — no `id` field. The list endpoint returns both unless filtered by `filter[type]`.

**Error-shape convention:** The `create-custom-currency` handler wraps ALL service errors as
`ConflictError`, but validation errors still resolve to `400` because `BaseAPIError.Unwrap()`
exposes the underlying `GenericValidationError` through the error encoder. Only genuine
duplicate-code conflicts return `409`. All other 4xx use `apierrors.GenericErrorEncoder()` → standard
`problem+json` shape.

**No lifecycle (no draft/publish/archive):** Custom currencies have no status transitions.
There is no delete endpoint — custom currencies persist indefinitely.

---

## Scenario list

**p0 — happy path**

- `currency_custom_lifecycle` — Create a custom currency, verify it appears in the list, filter by type, create a cost basis, list cost bases. — shape: lifecycle — priority: p0

**p1 — core validation**

- `currency_create_duplicate_code_rejected` — Creating a second custom currency with the same `code` returns 409 with detail containing `"Currency already exists"`. — shape: single-request — priority: p1
- `currency_create_missing_required_fields` — Creating a custom currency without `name` or `code` returns 400 (schema rule shape from OpenAPI binder). — shape: single-request — priority: p1
- `currency_create_symbol_required_by_domain` — Creating a custom currency without `symbol` returns 400 (domain validation; symbol is required by `CreateCurrencyInput.Validate()` even though the OpenAPI schema marks it optional). — shape: single-request — priority: p1 — NEEDS-VERIFY: confirm 400 vs pass-through when symbol absent
- `currency_cost_basis_rate_must_be_positive` — Creating a cost basis with `rate: "0"` or negative rate returns 400. — shape: single-request — priority: p1
- `currency_cost_basis_invalid_fiat_code` — Creating a cost basis with an unknown fiat code returns 400. — shape: single-request — priority: p1 — NEEDS-VERIFY: confirm error detail/code from server
- `currency_list_filter_by_type_custom` — `filter[type]=custom` returns only custom currencies (no fiat entries in `data[]`). — shape: single-request — priority: p1
- `currency_list_filter_by_type_fiat` — `filter[type]=fiat` returns only fiat currencies (no custom entry in `data[]`). — shape: single-request — priority: p1
- `currency_cost_bases_list_filter_by_fiat_code` — `filter[fiat_code]=USD` on the cost-bases list returns only cost bases with `fiat_code == "USD"`. — shape: single-request — priority: p1

**p2 — edge cases**

- `currency_list_pagination` — `page[number]` + `page[size]` return the expected slice and `meta.page.total`. — shape: single-request — priority: p2
- `currency_cost_basis_effective_from` — Cost basis created with an explicit `effective_from` timestamp returns that timestamp in the response. — shape: single-request — priority: p2
- `currency_cost_basis_nonexistent_currency` — Creating a cost basis for a non-existent `currencyId` returns 404 or 400. — shape: single-request — priority: p2 — NEEDS-VERIFY: confirm status code from server

---

## Baselines

### Baseline custom currency — `CreateCurrencyCustomRequest`

- `code`: unique per run (3–24 chars, must not conflict with ISO-4217 codes)
- `name`: `"Test Currency"`
- `symbol`: `"TC"` (symbol is required by domain `Validate()` even though OpenAPI schema marks it optional)

### Baseline cost basis — `CreateCostBasisRequest`

- `fiat_code`: `"USD"` (valid ISO-4217 code)
- `rate`: `"1.5"` (positive decimal string)
- `effective_from`: omitted (server sets to now)

---

## Scenario: currency_custom_lifecycle

```yaml
id: currency_custom_lifecycle
endpoints:
  - POST /openmeter/currencies/custom
  - GET /openmeter/currencies
  - POST /openmeter/currencies/custom/{currencyId}/cost-bases
  - GET /openmeter/currencies/custom/{currencyId}/cost-bases
entities: [currency, cost-basis]
tags: [lifecycle, crud, p0]
```

**Intent:** A custom currency is created, appears in the list (filterable by type), and can have a cost basis attached and listed.

**Fixtures:**
- A `CreateCurrencyCustomRequest` per **Baseline custom currency**.
- A `CreateCostBasisRequest` per **Baseline cost basis**.

**Steps:**

1. **Create custom currency.** `POST /openmeter/currencies/custom` with the fixture.
   - Expect `201 Created`.
   - Expect `id` is non-null.
   - Expect `type` is `"custom"`.
   - Expect `code` equals the fixture code.
   - Expect `name` equals the fixture name.
   - Expect `symbol` equals the fixture symbol.

   Captures:
   - `currency` ← `response.body`

2. **List all currencies — verify custom currency appears.**
   `GET /openmeter/currencies?page[size]=1000`.
   - Expect `200 OK`.
   - Expect `data[]` contains an entry with `id == {currency.id}`.

3. **List filtered by type `custom` — only custom currencies returned.**
   `GET /openmeter/currencies` with `filter[type]=custom` and `page[size]=1000`.
   - Expect `200 OK`.
   - Expect `data[]` contains an entry with `id == {currency.id}`.
   - Expect every item in `data[]` has `type == "custom"`.

4. **List filtered by type `fiat` — custom currency not in results.**
   `GET /openmeter/currencies` with `filter[type]=fiat` and `page[size]=1000`.
   - Expect `200 OK`.
   - Expect `data[]` does NOT contain an entry with `id == {currency.id}`.
   - Expect every item in `data[]` has `type == "fiat"`.

5. **Create cost basis.** `POST /openmeter/currencies/custom/{currency.id}/cost-bases`
   with the cost basis fixture.
   - Expect `201 Created`.
   - Expect `id` is non-null.
   - Expect `fiat_code` equals `"USD"`.
   - Expect `rate` equals `"1.5"`.

   Captures:
   - `cost_basis` ← `response.body`

6. **List cost bases.** `GET /openmeter/currencies/custom/{currency.id}/cost-bases`.
   - Expect `200 OK`.
   - Expect `data[]` contains an entry with `id == {cost_basis.id}`.

**Notes:**

- There is no DELETE endpoint for custom currencies — they persist indefinitely. No cleanup step.
- `symbol` is required by the domain `Validate()` even though the OpenAPI schema marks it optional in `CreateCurrencyCustomRequest`. Always include `symbol` in create requests.
- The create handler wraps ALL errors as `ConflictError`. Validation errors (missing fields, etc.) still resolve to `400` because the error encoder unwraps through `BaseAPIError.Unwrap()`. Only genuine duplicate-code conflicts return `409`.
- Fiat currencies returned from `GET /openmeter/currencies` have no `id` field in the response body (`id` is absent, not null).
