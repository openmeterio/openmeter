# httpdriver

<!-- archie:ai-start -->

> HTTP transport layer for the plan API: builds httptransport handlers (List/Create/Get/Update/Delete/Publish/Archive/Next) that decode requests, map API<->domain types, and delegate to plan.Service. Pure transport — no DB, no business rules.

## Patterns

**httptransport handler-per-operation with type aliases** — Each operation defines Request/Response/Handler type aliases (Request usually = plan.<Input>) and a method returning a httptransport.NewHandler(WithArgs) triple of decode/exec/encode. Service is the only dependency invoked in the exec step. (`type CreatePlanHandler httptransport.Handler[CreatePlanRequest, CreatePlanResponse]`)
**Handler interface aggregation + New constructor** — handler implements the Handler interface (PlanHandler with one method per operation). New(namespaceDecoder, service, credits, featureGate, options...) wires deps; var _ Handler = (*handler)(nil) enforces completeness. (`func New(namespaceDecoder namespacedriver.NamespaceDecoder, service plan.Service, credits appconfig.CreditsConfiguration, featureGate featuregate.Gate, options ...httptransport.HandlerOption) Handler`)
**FromX / AsX mapping convention in mapping.go** — FromPlan/FromPlanPhase map domain->api.*; AsCreatePlanRequest/AsUpdatePlanRequest/AsPlanPhase map api.*->domain. RateCard and metadata mapping is delegated to the shared productcatalog/http package (http.FromRateCard, http.AsRateCards, http.FromMetadata). (`func FromPlan(p plan.Plan) (api.Plan, error)`)
**Namespace resolution + ValidationErrorEncoder per handler** — Every decode step calls h.resolveNamespace(ctx) (500 if missing) before mapping; every handler appends httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(ResourceKindPlan)) and WithOperationName. (`httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan))`)
**Credits feature-gating for CreditOnly settlement** — featuregate.go isCreditsEnabled(ns) checks credits.Enabled, then the optional featuregate.Gate against credits.FeatureFlag. Create/Update reject CreditOnlySettlementMode with a generic validation error when credits are disabled. (`if !creditEnabled && req.SettlementMode == productcatalog.CreditOnlySettlementMode { return ..., models.NewGenericValidationError(...) }`)
**ParseIDOrKey for IDOrKey path params** — Get/Next accept an IDOrKey path arg and use ref.ParseIDOrKey to split into NamespacedID.ID vs Key (ULID detection), so one route serves both lookups. (`idOrKey := ref.ParseIDOrKey(params.IDOrKey)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Handler/PlanHandler interfaces, handler struct, New constructor, resolveNamespace | Adding an operation requires extending the PlanHandler interface or var _ Handler check fails to compile. |
| `plan.go` | All per-operation handlers (decode/exec/encode) and Request/Response type aliases | Publish/Archive/Next set EffectivePeriod from clock.Now() in the driver (TODO notes spec lacks the field); Create/Update set IgnoreNonCriticalIssues=true. |
| `mapping.go` | FromPlan/FromPlanPhase and AsCreate/AsUpdate/AsPlanPhase converters | asProRatingConfig defaults to Enabled=true + ProRatingModeProratePrices when nil; status mapping returns error on unknown PlanStatus. |
| `featuregate.go` | isCreditsEnabled gate combining credits config and featuregate.Gate | Returns true (gate-open) when gate is nil or FeatureFlag empty but credits.Enabled — only a configured flag actually evaluates per-namespace. |

## Anti-Patterns

- Putting business logic (status transitions, version policy) in handlers instead of plan.Service
- Bypassing resolveNamespace and reading the namespace ad hoc
- Hand-mapping RateCards/metadata instead of reusing productcatalog/http helpers
- Omitting WithErrorEncoder(ValidationErrorEncoder(ResourceKindPlan)), so validation errors serialize as generic 500s
- Allowing CreditOnlySettlementMode through without the isCreditsEnabled guard

## Decisions

- **Request types alias the service Input structs (CreatePlanRequest = plan.CreatePlanInput)** — Avoids a redundant transport DTO layer; mapping functions populate the same struct the service consumes.
- **Effective period for publish/archive is server-set from clock.Now()** — TypeSpec does not yet expose EffectivePeriod on these requests, so the driver supplies it (tracked by TODO comments).

## Example: An operation handler with namespace resolution and validation encoder

```
func (h *handler) CreatePlan() CreatePlanHandler {
  return httptransport.NewHandler(
    func(ctx context.Context, r *http.Request) (CreatePlanRequest, error) {
      body := api.PlanCreate{}
      if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil { return CreatePlanRequest{}, err }
      ns, err := h.resolveNamespace(ctx)
      if err != nil { return CreatePlanRequest{}, err }
      req, err := AsCreatePlanRequest(body, ns)
      // ... isCreditsEnabled guard, set IgnoreNonCriticalIssues ...
      return req, err
    },
    func(ctx context.Context, request CreatePlanRequest) (CreatePlanResponse, error) {
      p, err := h.service.CreatePlan(ctx, request)
      if err != nil { return CreatePlanResponse{}, err }
      return FromPlan(*p)
// ...
```

<!-- archie:ai-end -->
