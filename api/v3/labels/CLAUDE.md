# labels

<!-- archie:ai-start -->

> Bridges the api.Labels wire type (map[string]string) to domain models.Metadata and models.Annotations, enforcing key/value format rules and the 'openmeter_' annotation prefix convention. All v3 endpoints accepting or returning labels route through here.

## Patterns

**ToMetadataAnnotations for inbound labels** — Splits keys into Metadata (user keys) and Annotations (openmeter_-prefixed, prefix stripped). Validates every key/value; collects errors via errors.Join and wraps with NewNillableGenericValidationError. (`ma, err := labels.ToMetadataAnnotations(req.Body.Labels); if err != nil { return err }; input.Metadata = ma.Metadata`)
**FromMetadataAnnotations for outbound labels** — Builds *api.Labels from domain Metadata + Annotations. Annotation keys are re-prefixed with 'openmeter_'; annotation values that are not string/Stringer/TextMarshaler are silently skipped, as are keys failing ValidateLabel. (`resp.Labels = labels.FromMetadataAnnotations(entity.Metadata, entity.Annotations)`)
**ValidateLabel / ValidateLabels for format + reserved-prefix checks** — Validates against keyValueFormat regexp and reservedPrefixMatcher; returns models.ValidationIssue errors carrying WithHTTPStatusCodeAttribute(400) so they render as 400 through the error encoder. (`if err := labels.ValidateLabels(*req.Labels); err != nil { return err }`)
**openmeter_ prefix as the annotation namespace** — API keys starting with 'openmeter_' map to models.Annotations (prefix stripped); all others map to models.Metadata. User-supplied 'openmeter_' keys are rejected to prevent spoofing internal annotations. (`// {"env":"prod","openmeter_region":"us"} -> Metadata{env:prod}, Annotations{region:us}`)
**FromMetadata / ToMetadata convenience helpers** — FromMetadata[T ~map[string]string] and ToMetadata wrap the annotation-aware converters for entities that only carry Metadata (no Annotations). (`func ToMetadata(labels *api.Labels) (models.Metadata, error) { m, err := ToMetadataAnnotations(labels); return m.Metadata, err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `convert.go` | ToMetadataAnnotations (inbound) and FromMetadataAnnotations (outbound) plus FromMetadata/ToMetadata helpers; defines AnnotationsPrefix = "openmeter_". | FromMetadataAnnotations handles Stringer, TextMarshaler, and string annotation values; other types (and failing MarshalText) are silently skipped with no error. |
| `validate.go` | ValidateLabel / ValidateLabels using compiled keyValueFormat and reservedPrefixMatcher regexps; issues carry WithHTTPStatusCodeAttribute(400). | Reserved prefixes include openmeter, kong, konnect, insomnia, mesh, kic, kuma, and a leading underscore — user keys with these are always rejected, even outside annotation contexts. |

## Anti-Patterns

- Accepting api.Labels in a domain service input — domain types use models.Metadata and models.Annotations.
- Skipping ValidateLabel when writing annotation keys from external input — invalid keys silently disappear in FromMetadataAnnotations.
- Checking the 'openmeter_' prefix directly in handler code instead of calling ToMetadataAnnotations.
- Relying on annotation values that are not string/Stringer/TextMarshaler — they are silently dropped on output.

## Decisions

- **The openmeter_ prefix splits labels into metadata vs annotations at the API boundary.** — Domain code uses typed Annotations (any values) while the wire type is map[string]string; the prefix acts as a namespace marker surviving the round-trip without a separate wire field.

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
