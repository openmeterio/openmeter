# entitlement

<!-- archie:ai-start -->

> Access-control domain. The root declares entitlement.Service, EntitlementRepo, the Access model, the Entitlement/GenericProperties domain type with its three subtypes (metered/static/boolean), scheduling semantics (ActiveFrom/ActiveTo), usage-period logic, and versioned v2 events. Subpackages implement the metered connector, balance worker, snapshots and HTTP drivers.

## Patterns

**Entitlement subtype via SubTypeConnector** — EntitlementType (metered|static|boolean) selects a SubTypeConnector whose BeforeCreate maps CreateEntitlementInputs + feature.Feature into CreateEntitlementRepoInputs and whose GetValue computes the EntitlementValue (HasAccess()). (`type SubTypeConnector interface { GetValue(ctx, *Entitlement, time.Time) (EntitlementValue, error); BeforeCreate(CreateEntitlementInputs, feature.Feature) (*CreateEntitlementRepoInputs, error); AfterCreate(...) error }`)
**Scheduling via ActiveFrom/ActiveTo with cadence** — GenericProperties carries optional ActiveFrom/ActiveTo; ActiveFromTime() defaults to CreatedAt, ActiveToTime() falls back to DeletedAt; Entitlement implements models.CadenceComparable. IsActive(at) checks deletion, schedule window, and zero-length windows. (`var _ models.CadenceComparable = Entitlement{}
func (e Entitlement) GetCadence() models.CadencedModel { ... }`)
**MeasureUsageFrom dual source** — MeasureUsageFromInput resolves either FromTime(t) or FromEnum(CURRENT_PERIOD_START|NOW, currPeriod, now); the enum validates against its Values() set. (`m.FromEnum(MeasureUsageFromCurrentPeriodStart, currPer, now)`)
**Versioned literal event payloads** — events.go types out entitlementEventV2EntitlementLiteral field-by-field (instead of versioning the domain model) with mapEntitlementToV2 / ToDomainEntitlement round-trips; events are EntitlementCreatedEventV2 / EntitlementDeletedEventV2 with metadata Subject = customer resource path. (`entitlementCreatedEventV2Name = metadata.GetEventName(metadata.EventType{Subsystem: EventSubsystem, Name: "entitlement.created", Version: "v2"})`)
**Repository keys on customer + feature + time** — EntitlementRepo resolves active and scheduled entitlements by (namespace, customerID, featureKey, at); deactivation sets activeTo. Service.GetEntitlementOfCustomerAt disambiguates id-vs-featureKey by trying ID first. (`GetActiveEntitlementOfCustomerAt(ctx, namespace, customerID, featureKey, at) (*Entitlement, error)`)
**Grants only on metered entitlements** — CreateEntitlementGrantInputs embeds credit.CreateGrantInput; ErrEntitlementGrantsOnlySupportedForMeteredEntitlements (a ValidationIssue) gates grant attachment. (`var ErrEntitlementGrantsOnlySupportedForMeteredEntitlements = models.NewValidationIssue(...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement.go` | Entitlement/GenericProperties model, EntitlementType enum, CreateEntitlementInputs (+Equal/+Validate), IsActive/GetCadence, MeasureUsageFromInput | CreateEntitlementInputs.Equal is hand-written field-by-field; add new fields there too. Static type requires valid-JSON Config. |
| `connector.go` | Service interface (Create/Override/Schedule/Supersede/Get/List/GetAccess) and ListEntitlementsParams | GetEntitlementOfCustomerAt resolves ambiguous id-or-featureKey; features whose keys look like entitlement IDs are forbidden. |
| `entitlement_types.go` | EntitlementValue (HasAccess), NoAccessValue, SubTypeConnector seam | Each type routes through a SubTypeConnector; BeforeCreate errors must abort creation. |
| `repository.go` | EntitlementRepo interface (active/scheduled lookups, deactivate, usage-period upsert, namespace listing) | Time-windowed methods rely on ActiveFrom/ActiveTo + DeletedAt; respect Highwatermark/Cursor for expiry sweeps. |
| `events.go` | Versioned v2 entitlement event literals + domain round-trip | Event payload is a typed literal, NOT the domain struct; keep mapEntitlementToV2Literal and ToDomainEntitlement in sync. |
| `access.go` | Access = map of featureKey -> EntitlementValueWithId (Type/Value/ID) | GetAccess returns per-feature values keyed by featureKey. |
| `errors.go` | Typed errors (AlreadyExists/AlreadyDeleted/NotFound/WrongType/Forbidden) + grant/property-mismatch ValidationIssues | Use the ValidationIssue variants for API-surfaced create-mismatch / grant constraints. |

## Anti-Patterns

- Adding a field to CreateEntitlementInputs without updating its hand-written Equal and Validate.
- Bypassing the SubTypeConnector to compute an EntitlementValue or build repo inputs for a specific type.
- Attaching grants to non-metered entitlements (use ErrEntitlementGrantsOnlySupportedForMeteredEntitlements).
- Deriving active state from DeletedAt alone instead of IsActive(at)/GetCadence, which also account for ActiveFrom/ActiveTo and zero-length windows.
- Editing the domain Entitlement struct expecting events to follow — events use a separate typed v2 literal that must be updated explicitly.

## Decisions

- **Entitlement events are versioned as hand-typed literals instead of versioning the domain model.** — Versioning the full domain model was too large a lift; literal events let payload shape evolve independently while ToDomainEntitlement reconstructs the domain type for validation.
- **Three entitlement types share one model and are differentiated by a SubTypeConnector.** — Keeps generic CRUD/scheduling uniform while letting metered (usage+credit), static (config), and boolean access compute values differently.

## Example: Computing active state across schedule and deletion

```
func (e Entitlement) IsActive(at time.Time) bool {
	if e.DeletedAt != nil && !at.Before(*e.DeletedAt) { return false }
	if e.ActiveFromTime().After(at) { return false }
	if e.ActiveTo != nil && !at.Before(*e.ActiveTo) { return false }
	if e.ActiveToTime() != nil && e.ActiveFromTime().Equal(*e.ActiveToTime()) { return false }
	return true
}
```

<!-- archie:ai-end -->
