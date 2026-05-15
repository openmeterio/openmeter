# labels

<!-- archie:ai-start -->

> Bridges the api.Labels wire type (map[string]string) to domain models.Metadata and models.Annotations, enforcing key/value format rules and the 'openmeter_' annotation prefix convention. All v3 endpoints that accept or return labels must route through this package.

## Patterns

**ToMetadataAnnotations for inbound label conversion** — When a v3 request body carries *api.Labels, call labels.ToMetadataAnnotations to split keys into Metadata (user keys) and Annotations (openmeter_* prefix stripped). Returns a validation error if any key violates format or reserved-prefix rules. (`ma, err := labels.ToMetadataAnnotations(req.Body.Labels)
if err != nil { return err }
input.Metadata = ma.Metadata`)
**FromMetadataAnnotations for outbound label conversion** — When building a v3 response from a domain entity, call labels.FromMetadataAnnotations(entity.Metadata, entity.Annotations) to produce *api.Labels. Annotation keys are re-prefixed with 'openmeter_'; annotation values that are not string/Stringer/TextMarshaler are silently skipped. (`resp.Labels = labels.FromMetadataAnnotations(entity.Metadata, entity.Annotations)`)
**ValidateLabel / ValidateLabels for pre-write validation** — Call ValidateLabel(k, v) or ValidateLabels(labels) to check format (^[a-zA-Z0-9][...]{1,63}$) and reserved prefixes. Returns models.ValidationIssue errors with http.StatusBadRequest attribute so they propagate through the error encoder. (`if err := labels.ValidateLabels(*req.Labels); err != nil { return err }`)
**openmeter_ prefix as the annotation namespace** — Keys starting with 'openmeter_' in API labels map to models.Annotations (with the prefix stripped); all other keys map to models.Metadata. The reserved-prefix validator rejects user-supplied 'openmeter_' keys to prevent spoofing internal annotations. (`// labels: {"env": "prod", "openmeter_region": "us"} →
// Metadata: {"env": "prod"}, Annotations: {"region": "us"}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `convert.go` | ToMetadataAnnotations (inbound) and FromMetadataAnnotations (outbound) converters plus simpler FromMetadata and ToMetadata helpers for entities that only use Metadata. | FromMetadataAnnotations handles Stringer, encoding.TextMarshaler, and string annotation values; types that don't implement any of these are silently skipped — no error is returned. |
| `validate.go` | ValidateLabel / ValidateLabels using two compiled regexps: keyValueFormat and reservedPrefixMatcher. ValidationIssues carry WithHTTPStatusCodeAttribute(400) so they render as 400 without extra mapping. | Reserved prefixes include 'openmeter' — user-supplied keys with this prefix are always rejected, even in non-annotation contexts. |

## Anti-Patterns

- Accepting api.Labels in a domain service input struct — domain types use models.Metadata and models.Annotations
- Skipping ValidateLabel when writing annotation keys from external input — invalid keys silently disappear in FromMetadataAnnotations
- Directly checking for the 'openmeter_' prefix in handler code instead of calling ToMetadataAnnotations
- Accepting annotation values that are not string/Stringer/TextMarshaler — they are silently dropped by FromMetadataAnnotations

## Decisions

- **openmeter_ prefix splits labels into metadata vs annotations at the API boundary** — Domain code works with typed Annotations (any values) while the API wire type is map[string]string; the prefix acts as a namespace marker that survives the round-trip without requiring a separate wire field.

## Example: Converting inbound labels in a create handler and outbound in a response mapper

```
import apiLabels "github.com/openmeterio/openmeter/api/v3/labels"

// Decode
ma, err := apiLabels.ToMetadataAnnotations(req.Body.Labels)
if err != nil { return err }
input := CreateInput{Metadata: ma.Metadata, Annotations: ma.Annotations}

// Encode
resp.Labels = apiLabels.FromMetadataAnnotations(entity.Metadata, entity.Annotations)
```

<!-- archie:ai-end -->
