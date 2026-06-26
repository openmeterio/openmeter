# grant

<!-- archie:ai-start -->

> Core grant domain model: the immutable Grant aggregate (amount, priority, effective/expiration, recurrence, rollover), its Repo interface, the OwnerConnector abstraction over the grant owner (entitlement), and versioned grant.created/grant.voided Watermill events.

## Patterns

**Effective-period derivation** — Grant.GetEffectivePeriod folds EffectiveAt/ExpiresAt with DeletedAt and VoidedAt into a StartBoundedPeriod, clamping To to the earliest cutoff and collapsing to a zero-length period when a grant never activates; ActiveAt delegates to it. (`func (g Grant) ActiveAt(t time.Time) bool { return g.GetEffectivePeriod().Contains(t) }`)
**Versioned events: v1 embeds, v2 uses literals** — Deprecated grantEventV1 embeds the Grant domain struct (CreatedEvent/VoidedEvent); v2 (CreatedEventV2/VoidedEventV2) uses grantEventV2GrantLiteral primitives with ToDomainGrant/mapGrantToV2Literal so domain changes don't break serialized events. (`func NewCreatedEventV2FromGrant(g Grant, c streaming.Customer) CreatedEventV2 { return CreatedEventV2{Namespace: ..., Grant: mapGrantToV2Literal(g), CustomerID: c.GetUsageAttribution().ID} }`)
**Repo is transaction-capable** — Repo embeds entutils.TxCreator and entutils.TxUser[Repo] alongside CRUD methods, so the adapter must implement Tx/WithTx/Self. (`type Repo interface { CreateGrant(...); ...; entutils.TxCreator; entutils.TxUser[Repo] }`)
**OwnerConnector resolves owner usage/reset semantics** — OwnerConnector abstracts the grant owner (a metered entitlement): DescribeOwner (Meter + DefaultQueryParams + StreamingCustomer), reset timeline/usage-period queries, EndCurrentUsagePeriod, and LockOwnerForTx. (`DescribeOwner(ctx, id) (Owner, error); GetResetTimelineInclusive(ctx, id, period) (timeutil.SimpleTimeline, error)`)
**Rollover/recurrence balance math on the model** — RolloverBalance clamps to [ResetMinRollover, ResetMaxRollover]; RecurrenceBalance returns the full Amount (no rollover for recurring grants); GetExpiration computes ExpiresAt from EffectiveAt+ExpirationPeriod. (`func (g Grant) RolloverBalance(b float64) float64 { return math.Min(g.ResetMaxRollover, math.Max(g.ResetMinRollover, b)) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `grant.go` | Grant aggregate (ManagedModel+NamespacedModel) and its methods: Validate, GetExpiration, GetEffectivePeriod, ActiveAt, RolloverBalance, RecurrenceBalance, GetNamespacedID/Owner. | Grant is documented as immutable; ExpiresAt is exclusive; Validate currently always returns nil; Priority is uint8 with lower = higher priority. |
| `owner_connector.go` | Owner struct, ResetBehavior, OwnerConnector interface, OwnerNotFoundError (+NewOwnerNotFoundError wrapping models.NewGenericNotFoundError). | GetResetTimelineInclusive's first returned time may precede the period start (documented semantics); both EndCurrentUsagePeriod and LockOwnerForTx mutate owner state. |
| `repo.go` | Repo interface, ListParams, RepoCreateInput, OrderBy enum + Values/StrValues. | Limit/Offset fields are marked deprecated in favor of Page; ListGrants returns a bare array for backward compat when pagination is absent. |
| `events.go` | Deprecated v1 events: grantEventV1 (embeds Grant), CreatedEvent, VoidedEvent, EventSubsystem='credit'. | Embedding the domain model means schema changes break old events — do not add new events here; use v2. |
| `events_2.go` | v2 events with primitive grantEventV2GrantLiteral, CreatedEventV2/VoidedEventV2, NewCreatedEventV2FromGrant/NewVoidedEventV2FromGrant. | v2 events carry CustomerID (from streaming.Customer usage attribution) and require it non-empty in Validate. |
| `expiration.go` | ExpirationPeriod (Count+Duration) and GetExpiration; ExpirationPeriodDuration enum HOUR/DAY/WEEK/MONTH/YEAR. | Unknown duration returns a zero time.Time; WEEK is implemented as Count*7 days. |

## Anti-Patterns

- Adding new event types that embed the Grant domain struct (v1 pattern) — use primitive literals as in events_2.go.
- Mutating a Grant in place — it is an immutable definition; balance changes live in the engine/balance.Map.
- Computing active/expiry windows manually instead of GetEffectivePeriod (which clamps Deleted/Voided cutoffs).
- Implementing Repo without the embedded entutils.TxCreator/TxUser[Repo] transaction methods.

## Decisions

- **Two event versions coexist; v2 serializes only primitives plus CustomerID.** — v1 embedded the domain model, so any field change broke replayed events; v2 decouples wire format from the domain and adds customer attribution for the streaming layer.
- **Grant is immutable and owner-agnostic via OwnerConnector.** — Grants belong to metered entitlements but the credit package stays decoupled from entitlement internals by depending only on the OwnerConnector interface and streaming.Customer.

## Example: Deriving a grant's effective period with deletion/void clamping

```
func (g Grant) GetEffectivePeriod() timeutil.StartBoundedPeriod {
	p := timeutil.StartBoundedPeriod{From: g.EffectiveAt, To: g.ExpiresAt}
	if g.DeletedAt != nil {
		if p.To == nil || g.DeletedAt.Before(*p.To) { p.To = g.DeletedAt }
	}
	if g.VoidedAt != nil {
		if p.To == nil || g.VoidedAt.Before(*p.To) { p.To = g.VoidedAt }
	}
	if p.To != nil && p.To.Before(p.From) { p.To = &p.From }
	return p
}
```

<!-- archie:ai-end -->
