# models

<!-- archie:ai-start -->

> Shared, dependency-free value types embedded inside event payloads across credit, entitlement, notification, and sink consumers — currently FeatureKeyAndID and NamespaceID, each carrying a Validate() error method. It exists so multiple event packages reference the same serialized shapes instead of redefining them.

## Patterns

**Validatable value type** — Each type is a small JSON-tagged struct with a value-receiver Validate() error that returns errors.New on the first missing required field. New types here follow the same shape. (`func (i NamespaceID) Validate() error { if i.ID == "" { return errors.New("namespace-id is required") }; return nil }`)
**JSON-stable field tags** — Every field has an explicit json tag (`key`, `id`) because these structs are serialized into the event bus; the tags are part of the wire format. (`type FeatureKeyAndID struct { Key string `json:"key"`; ID string `json:"id"` }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Holds all shared event payload value types: FeatureKeyAndID (feature key+id pair) and NamespaceID (namespace id wrapper), each with Validate(). | Validate() uses plain errors.New and stops at the first invalid field — it is not a NewNillableGenericValidationError aggregate like domain Validate() methods elsewhere; keep that lightweight style here. Renaming json tags breaks deserialization of already-published events. |

## Anti-Patterns

- Importing domain/service packages here — this must remain a zero-dependency leaf (only `errors`) to avoid import cycles with its many event-consumer importers.
- Changing existing json tags or field semantics, which silently breaks decoding of in-flight events.
- Adding behavior beyond plain data + Validate(); business logic belongs in the owning domain package.

## Decisions

- **Co-locate cross-cutting event payload structs in one dependency-free package.** — Avoids each event producer/consumer redefining incompatible copies and prevents import cycles given the wide importer set (credit/grant, entitlement, notification/consumer, sink ingestnotification).

## Example: Embed and validate a shared payload type in an event

```
import "github.com/openmeterio/openmeter/openmeter/event/models"

type FeatureCreatedEvent struct {
    Namespace models.NamespaceID     `json:"namespace"`
    Feature   models.FeatureKeyAndID `json:"feature"`
}

func (e FeatureCreatedEvent) Validate() error {
    if err := e.Namespace.Validate(); err != nil { return err }
    return e.Feature.Validate()
}
```

<!-- archie:ai-end -->
