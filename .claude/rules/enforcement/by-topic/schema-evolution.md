# Enforcement: schema-evolution (1 rule)

Topic file. Loaded on demand when an agent works on something in the `schema-evolution` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pitfalls (block)

### `pf-011-billing-schema-level-dualwrite` — Do not remove BillingInvoice deprecated columns, the schema_level field, or the BillingInvoiceWriteSchemaLevel table until the in-flight invoice-line migration is complete; deprecated columns and the schema_level dual-write must be removed in lockstep across the Ent schema and the billing adapter.

*source: `deep_scan`*

**Why:** Pitfall pf_0011: an in-progress billing-invoice schema migration leaves deprecated columns and temporary versioning artifacts live (billing.go:416 line_ids Deprecated; billing.go:631 BillingInvoiceSplitLineGroup tax_config Deprecated; billing.go:1170 schema_level Default(1); billing.go:1360 BillingInvoiceWriteSchemaLevel temporary single-row table). The migration is mid-flight, so deprecated columns and the schema_level dual-write coexist; removing one side without the other corrupts invoice reads/writes.

**Example:**

```
// Deprecated, not dropped — keep until migration completes:
field.String("line_ids").Optional().Nillable().Deprecated("invoice discounts are deprecated, use line_discounts instead")
```

**Path glob:** `openmeter/ent/schema/billing.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "schema_level|BillingInvoiceWriteSchemaLevel|Deprecated\\("
    ]
  }
]
```

</details>
