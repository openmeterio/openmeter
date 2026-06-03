# currencies

<!-- archie:ai-start -->

> v3 HTTP handlers for listing/creating custom currencies and cost bases; resolves namespace via namespacedriver.NamespaceDecoder and delegates to currencies.CurrencyService.

## Patterns

**NamespaceDecoder instead of resolveNamespace closure** — This package holds a namespacedriver.NamespaceDecoder and calls h.namespaceDecoder.GetNamespace(ctx) returning (string, bool) — not a func(ctx)(string,error). A false ok returns apierrors.NewInternalError. (`ns, ok := h.namespaceDecoder.GetNamespace(ctx); if !ok { return CreateCurrencyRequest{}, apierrors.NewInternalError(ctx, fmt.Errorf("failed to resolve namespace")) }`)
**Page default size differs by operation** — ListCurrencies defaults to page size 100 (currency catalog) while ListCostBases defaults to 20. (`page := pagination.NewPage(1, 100)`)
**BillingCurrency union via generic constructor** — convert.go builds the v3.BillingCurrency discriminated union with NewBillingCurrencyFrom[T] calling the generated FromBillingCurrencyCustom/FromBillingCurrencyFiat methods. (`func NewBillingCurrencyFrom[T v3.BillingCurrencyCustom | v3.BillingCurrencyFiat](v T) (v3.BillingCurrency, error)`)
**Custom-vs-fiat routing by empty ID** — ToAPIBillingCurrency treats any currency with c.ID == "" as fiat and otherwise custom. (`if c.ID != "" { return NewBillingCurrencyFrom(v3.BillingCurrencyCustom{...}) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface (ListCurrencies, CreateCurrency, CreateCostBasis, ListCostBases) + handler struct holding namespaceDecoder, currencies.CurrencyService, options. | Uses namespacedriver.NamespaceDecoder, not a closure; new ops must call GetNamespace (returns bool, not error). |
| `list.go` | ListCurrencies with optional filter[type] and filter[code], sort parsing via request.ParseSortBy, page default 100. | params.Filter is optional — nil-check before dereferencing Type/Code. |
| `convert.go` | FromAPIBillingCurrencyType, NewBillingCurrencyFrom[T], ToAPIBillingCurrency, ToAPIBillingCostBasis (Rate Decimal -> string). | Custom vs fiat is decided solely by c.ID != ""; an empty ID is always treated as fiat. |
| `get_cost_bases.go` | ListCostBases with a custom ListCostBasesArgs (CurrencyID + Params) and optional filter[fiat_code]. | Default page size here is 20 (not 100); FilterFiatCode is built only when Params.Filter.FiatCode is non-nil. |

## Anti-Patterns

- Using the resolveNamespace closure pattern instead of namespaceDecoder.GetNamespace
- Defaulting ListCurrencies page size to 20 (use 100 for the currency catalog)
- Mixing currency/cost-basis conversion logic into operation files instead of convert.go
- Ignoring the ok bool from GetNamespace and assuming a valid namespace

## Decisions

- **namespacedriver.NamespaceDecoder is injected instead of a resolveNamespace closure** — currencies was wired before the closure pattern was standardized; both resolve namespace from context, the decoder pattern is the older convention.

<!-- archie:ai-end -->
