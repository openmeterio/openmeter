# topicresolver

<!-- archie:ai-start -->

> Provides the Resolver interface and its single implementation NamespacedTopicResolver, which maps a namespace string to a Kafka topic name using a printf-style template (e.g. 'om_%s_events'). Used by the Kafka ingest collector to determine the target topic per ingest call.

## Patterns

**Resolver interface** — All topic resolution must go through the Resolver interface: Resolve(ctx context.Context, namespace string) (string, error). Callers must never hard-code topic names. (`type Resolver interface { Resolve(ctx context.Context, namespace string) (string, error) }`)
**Template-based namespace-to-topic mapping** — NamespacedTopicResolver uses fmt.Sprintf(template, namespace). Template must contain exactly one %s placeholder. Constructor validates non-empty template. (`NewNamespacedTopicResolver("om_%s_events")`)
**Compile-time interface assertion** — var _ Resolver = (*NamespacedTopicResolver)(nil) ensures the concrete type satisfies the interface at compile time. (`var _ Resolver = (*NamespacedTopicResolver)(nil)`)
**Context threading** — Resolve accepts context.Context even though NamespacedTopicResolver ignores it. New implementations (e.g. DB-backed resolvers) may use it for cancellation — always pass the caller's ctx. (`func (r NamespacedTopicResolver) Resolve(_ context.Context, namespace string) (string, error)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `resolver.go` | Defines only the Resolver interface. New resolver implementations belong in separate files in this package. | Do not add business logic to resolver.go — keep it as a pure interface definition. |
| `namespacedtopic.go` | Single-template resolver; constructor rejects empty templates. The only current Resolver implementation. | Template must have exactly one %s; passing a template with zero or two %s arguments causes a runtime fmt.Sprintf format error — constructor does not validate placeholder count. |

## Anti-Patterns

- Hard-coding topic names in the collector instead of using Resolver.Resolve
- Passing a template string with zero or multiple %s placeholders to NewNamespacedTopicResolver (constructor only validates non-empty, not placeholder count)
- Implementing Resolver without accepting context.Context — the interface requires it for future async/DB-backed implementations
- Calling Resolve with context.Background() in application code — always propagate the caller's context

## Decisions

- **Topic name is derived from a template rather than stored in config as a static string** — Enables per-namespace Kafka topic isolation (multi-tenant ingest) without requiring config changes per new namespace
- **Resolver is an interface, not a function type** — Allows future implementations (e.g. DB-backed dynamic routing) to be swapped in via DI without changing the collector

<!-- archie:ai-end -->
