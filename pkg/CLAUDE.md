# pkg

<!-- archie:ai-start -->

> Domain-agnostic shared Go utilities — the foundational layer every openmeter/* domain, api/*, cmd/*, and app/* package builds on. Its primary constraint: code here must NOT import openmeter/* domain packages, so it stays a pure dependency sink (pkg/models alone has ~229 in-edges).

## Patterns

**Validate() aggregates, never short-circuits** — Validate() methods collect into `var errs []error`, errors.Join them, and return models.NewNillableGenericValidationError(...). Established in pkg/models and mirrored by pkg/currencyx, pkg/filter, pkg/expand, pkg/timeutil. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**Read time via clock.Now()** — Time-dependent utilities (pkg/lrux expiry, period/cadence math) read pkg/clock.Now() not time.Now(), so clock.FreezeTime/UnFreeze make tests deterministic. (`import "github.com/openmeterio/openmeter/pkg/clock"; now := clock.Now()`)
**Config struct with Validate + Configure(viper) + New constructor** — Infra packages (pkg/redis, pkg/kafka, pkg/pglockx) expose a Config with Validate(), viper Configure(), and a validating New* constructor; direct client construction that skips OTel/validation is forbidden. (`client, err := cfg.NewClient(...) // not redis.NewClient(rawOpts) directly`)
**Inject *slog.Logger, never slog.Default()** — pkg/errorsx, pkg/gosundheit, pkg/log, pkg/kafka require an explicitly injected logger; falling back to slog.Default() in a constructor is a project-wide anti-pattern. (`func NewSlogHandler(logger *slog.Logger) Handler`)
**Generic value helpers wrap samber/lo, not replace it** — pkg/slicesx, pkg/convert, pkg/defaultx, pkg/set complement lo with nil-preserving / empty-aware / error-aware variants; only add a local wrapper when lo cannot express it (e.g. convert.MapToPointer vs lo.EmptyableToPtr). (`ptr := convert.MapToPointer(m) // lo.EmptyableToPtr mishandles maps`)
**Cycle-break by duplication, not cross-import** — pkg/equal duplicates the Equaler[T] interface from pkg/models specifically to avoid an import cycle; foundational packages stay maximally importable by holding only generic primitives. (`// pkg/equal: Equaler[T] duplicated, must NOT import pkg/models`)
**Single-purpose leaf packages** — Most children own one tightly-scoped concern (pkg/cmpx, pkg/hasher, pkg/idempotency, pkg/strcase, pkg/sortx). Resist adding unrelated logic; grow a new sibling package instead. (`pkg/sortx holds only the ASC/DESC Order enum shared codebase-wide`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pkg/models` | Most-depended-on package (~229 in-edges): ManagedModel/NamespacedModel/CadencedModel, the Generic*Error taxonomy, and the ValidationIssue/FieldDescriptor RFC-7807 mapping. Signatures here are effectively frozen public API. | ErrorSeverity is inverted (Critical is the LOWEST value); never set ValidationIssue fields directly — use the With* option constructors which Clone. |
| `pkg/framework` | Structural sub-namespace owning transaction/entutils (the adapter transaction foundation), the operation->httptransport->commonhttp HTTP stack, lockr, pgdriver, tracex. | entutils.TransactingRepo/TxDriver is the ONLY sanctioned shared-transaction path — never reach to Ent's native client.Tx; framework must not import any openmeter/* domain package. |
| `pkg/pagination` | v1 offset/limit primitives (Page 1-based, Result[T], NewPaginator closure, CollectAll) used by ~119 packages; cursor/keyset lives in pkg/pagination/v2. | PageNumber is 1-based (Offset() goes negative if treated as 0-based); CollectAll returns nil (not partial) on error. |
| `pkg/timeutil` | Period/Recurrence/Timeline algebra used by billing, subscription, entitlement, credit; immutable value receivers delegating calendar math to pkg/datetime. | OpenPeriod.Intersection==nil means NO overlap (not empty); use Recurrence not raw time.Add so variable-length months don't overflow. |
| `pkg/datetime` | DateTime (RFC9557 time.Time wrapper) and ISODuration over rickb777/period; no-overflow calendar arithmetic via shiftClockTo. | Use DateTime.Add* not time.AddDate (month-end overflow); MarshalJSON serializes RFC3339 and drops bracket timezone/nanos. |
| `pkg/filter` | AIP-style typed query filters (FilterString/Integer/Time/...) emitting BOTH Ent predicates and go-sqlbuilder WHERE expressions; ~53 in-edges from v3 handlers/adapters. | An operator added to Select but not SelectWhereExpr (or vice versa) silently diverges the two SQL emitters; escape LIKE input via EscapeLikePattern. |
| `pkg/kafka` | Typed librdkafka Consumer/Producer/Admin config builders + LRU-cached idempotent TopicProvisioner; consumed by ingest, sink, watermill, cmd/*. | Every config struct needs `var _ ConfigMapper`/`var _ ConfigValidator`; guard SetKey on non-zero or you override librdkafka defaults; tolerate ErrTopicAlreadyExists. |
| `pkg/currencyx` | Currency Code (ISO 4217, used directly in the Ent schema), Calculator (resolves precision once), largest-remainder amount allocation over gobl. | Calculator{} constructed directly has nil Def and panics on RoundToPrecision; round to subunits via RoundToPrecision (JPY has 0 subunits), not decimal.Round. |

## Anti-Patterns

- Importing any openmeter/* domain package from anywhere under pkg/ — it must remain a pure dependency sink so every domain can depend on it.
- Calling time.Now() instead of clock.Now() in time-dependent utilities, breaking deterministic FreezeTime tests.
- Falling back to slog.Default() in a constructor instead of requiring an injected *slog.Logger.
- Adding local pointer/slice/must/default wrappers when github.com/samber/lo already covers the need (pkg/convert's empty-aware container helpers are the sanctioned exception).
- Returning on the first validation error instead of joining all issues into models.NewNillableGenericValidationError.

## Decisions

- **pkg is a flat collection of single-purpose, domain-agnostic packages rather than a layered library.** — Keeping each utility narrow and free of domain imports maximizes reuse and lets 200+ packages depend on them without import cycles.
- **pkg/framework concentrates the cross-cutting transaction, HTTP-operation, locking, and driver foundations as its own sub-namespace.** — These sit at the bottom of the dependency graph; isolating them as domain-agnostic foundations means every domain's service/adapter layer can build on one sanctioned transaction and HTTP-handler abstraction.
- **pkg/equal duplicates pkg/models' Equaler rather than sharing it.** — Deliberate duplication breaks an import cycle so equality-based diffing stays usable from packages that pkg/models would otherwise depend on.

## Example: The project-standard Validate() that aggregates all field issues

```
import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

func (i Input) Validate() error {
	var errs []error
	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}
// ...
```

<!-- archie:ai-end -->
