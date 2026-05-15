# Enforcement: security (1 rule)

Topic file. Loaded on demand when an agent works on something in the `security` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pattern Divergence (inform)

### `portal-001` — Portal token meter-slug allowlist validation must happen in the HTTP handler against meter.Service, not inside portal.Service itself.

*source: `deep_scan`*

**Why:** The openmeter/portal component description states: 'Meter slug validation happens in the HTTP handler operation against meter.Service, not inside portal.Service itself.' Portal.Service only issues and validates JWT tokens scoped to namespace, subject, and optional meter slug allowlist. Placing meter existence checks inside portal.Service creates a cross-domain import (portal imports meter) that violates the layered boundary.

**Example:**

```
// Correct: HTTP handler validates meter slugs before calling portal.Service.CreateToken
for _, slug := range req.Params.AllowedMeters {
    if _, err := meterSvc.GetMeterByIDOrSlug(ctx, ns, slug); err != nil {
        return nil, models.NewGenericNotFoundError("meter not found")
    }
}
return portalSvc.CreateToken(ctx, ...)

// Wrong: portal.Service.CreateToken queries meter.Service internally
```

**Path glob:** `openmeter/portal/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "meter\\.Service|meter\\.GetMeter"
    ],
    "must_not_match": [
      "// meter validation",
      "httpdriver",
      "httphandler"
    ]
  }
]
```

</details>
