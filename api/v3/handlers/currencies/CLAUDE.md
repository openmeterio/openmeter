# currencies

<!-- archie:ai-start -->

> HTTP handlers for listing custom currencies and cost bases in the v3 API; uses namespacedriver.NamespaceDecoder (not a resolveNamespace closure) for namespace resolution and delegates to currencies.CurrencyService.

## Patterns

**NamespaceDecoder instead of resolveNamespace closure** — Unlike most handler packages that accept a resolveNamespace func(ctx) (string, error), this handler holds a namespacedriver.NamespaceDecoder and calls h.namespaceDecoder.GetNamespace(ctx) returning (string, bool). (`ns, ok := h.namespaceDecoder.GetNamespace(ctx); if !ok { return ..., apierrors.NewInternalError(ctx, fmt.Errorf("failed to resolve namespace")) }`)
**Page defaults differ by operation** — ListCurrencies uses default page size 100 (not 20 like other handlers), reflecting expected catalog size. (`page := pagination.NewPage(1, 100)`)
**Handler interface declares all resource sub-types** — The Handler interface includes both currency (ListCurrencies, CreateCurrency) and cost basis (CreateCostBasis, ListCostBases) endpoints in a single handler struct. (`type Handler interface { ListCurrencies() ...; CreateCurrency() ...; CreateCostBasis() ...; ListCostBases() ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface + handler struct holding namespaceDecoder, currencyService currencies.CurrencyService, and options. New() constructor. | This handler uses namespacedriver.NamespaceDecoder not a func; new operations must call GetNamespace not resolveNamespace. |
| `list.go` | ListCurrencies with optional filter[type] query param mapped through FromAPIBillingCurrencyType and page-pagination with default size 100. | Filter is optional; nil check required before dereferencing params.Filter. |

## Anti-Patterns

- Using resolveNamespace pattern instead of namespaceDecoder.GetNamespace
- Defaulting page size to 20 (use 100 for currency catalog lists)
- Mixing currency and cost-basis conversion logic into operation files rather than a convert.go

## Decisions

- **namespacedriver.NamespaceDecoder is injected instead of a resolveNamespace closure because currencies was wired before the closure pattern was standardized.** — Both patterns resolve namespace from context; the decoder pattern is older and specific to certain handler packages.

<!-- archie:ai-end -->
