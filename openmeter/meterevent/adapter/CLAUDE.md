# adapter

<!-- archie:ai-start -->

> Implements the meterevent.Service interface by delegating to streaming.Connector for raw event queries, then enriching results with customer IDs and validating events against meter definitions. This is the single service implementation — not a DB adapter; it composes three injected services (streaming, customer, meter).

## Patterns

**Interface compliance assertion** — Declare `var _ meterevent.Service = (*adapter)(nil)` at the top of adapter.go to guarantee compile-time conformance. (`var _ meterevent.Service = (*adapter)(nil)`)
**Validate-before-delegate** — Both ListEvents and ListEventsV2 call params.Validate() and wrap the error in models.NewGenericValidationError before calling the streaming connector. (`if err := params.Validate(); err != nil { return nil, models.NewGenericValidationError(fmt.Errorf("validate input: %w", err)) }`)
**Post-process pipeline** — After fetching raw events, always call eventPostProcess (mapEventsToMeterEvents → enrichEventsWithCustomerID → validateEvents). Never short-circuit this pipeline. (`meterEvents, err = a.eventPostProcess(ctx, params.Namespace, rawEvents)`)
**Early-empty-return on customer resolution** — When customerIDs filter is provided but no matching customers exist, return an empty slice immediately rather than querying the streaming layer. (`if len(customers) == 0 { return []meterevent.Event{}, nil }`)
**Subject→customerID cache** — enrichEventsWithCustomerID uses a per-call map[string]string cache to avoid repeated DB lookups for the same subject. New enrichment helpers must follow this pattern. (`cache := make(map[string]string); if customerID, ok := cache[event.Subject]; ok { ... }`)
**Cursor emission only on full page** — In ListEventsV2, emit NextCursor only when len(meterEvents) == effectiveLimit; otherwise leave it nil. (`if len(meterEvents) > 0 && len(meterEvents) == effectiveLimit { cursor := meterEvents[len(meterEvents)-1].Cursor(); result.NextCursor = &cursor }`)
**ValidationErrors attached to event, not returned as error** — validateEvents populates Event.ValidationErrors per event; it does NOT return an error for individual invalid events. The caller receives all events regardless. (`event.ValidationErrors = validationErrors; validatedEvents = append(validatedEvents, event)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Constructor and struct definition. New() returns meterevent.Service. Holds streamingConnector, customerService, meterService. | Do not add DB client here; this adapter never touches Ent/Postgres directly. |
| `event.go` | All method implementations: ListEvents, ListEventsV2, listCustomers, eventPostProcess, mapEventsToMeterEvents, validateEvents, enrichEventsWithCustomerID. | enrichEventsWithCustomerID has a FIXME: it calls GetCustomerByUsageAttribution per event. The cache mitigates N+1 only within a single call. Batching is a known TODO. |

## Anti-Patterns

- Importing openmeter/ent/db or any Ent-generated package — this adapter is streaming-only.
- Returning an error from validateEvents for individual event failures — attach to Event.ValidationErrors instead.
- Skipping params.Validate() before delegating to the streaming connector.
- Emitting NextCursor when the result page is smaller than the effective limit.
- Calling customerService or meterService inside the hot loop without caching (already flagged as FIXME for customerService).

## Decisions

- **Service is implemented as an adapter/ sub-package rather than a service/ sub-package.** — meterevent has no separate business logic layer; the 'service' is purely a composition of streaming queries + customer/meter lookups, so the adapter pattern suffices without a distinct service struct.
- **ValidationErrors are per-event fields, not returned as a top-level error.** — Event listing is a query API; callers need to see all events including partially-invalid ones (e.g. unknown meter type). Failing the whole request for one bad event would break observability workflows.

## Example: Adding a new filter that resolves IDs from a domain service before querying the streaming layer

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) ListEventsV2(ctx context.Context, params meterevent.ListEventsV2Params) (...) {
	if err := params.Validate(); err != nil {
		return ..., models.NewGenericValidationError(fmt.Errorf("validate input: %w", err))
	}
	listParams := streaming.ListEventsV2Params{ /* map fields */ }
	if params.CustomerID != nil && len(*params.CustomerID.In) > 0 {
// ...
```

<!-- archie:ai-end -->
