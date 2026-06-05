# hooks

<!-- archie:ai-start -->

> Structural folder for subscription command hooks — cross-cutting side effects that run around subscription mutations. Currently holds only annotations/, the hook that repairs the previous/superseding subscription cross-link annotations on delete.

## Patterns

**SubscriptionCommandHook via embedding** — Hooks embed subscription.NoOpSubscriptionCommandHook and override only the lifecycle method they need (e.g. BeforeDelete), keeping the full interface satisfied. (`type hook struct { subscription.NoOpSubscriptionCommandHook; ... }`)
**Constructor dependency validation** — Each hook's constructor checks non-nil deps and an injected *slog.Logger and returns an error rather than using slog.Default(). (`if logger == nil { return nil, errors.New("logger is required") }`)

## Anti-Patterns

- Implementing a hook without embedding NoOpSubscriptionCommandHook, breaking the subscription.SubscriptionCommandHook contract.
- Putting uniqueness/overlap validation here — that belongs in validators/, hooks are for side effects and link repair.
- Falling back to slog.Default() instead of requiring an injected logger.

## Decisions

- **Annotation-link cleanup runs as a BeforeDelete hook rather than inline in the service.** — Keeps the doubly-linked-list repair out of the core mutation path and reusable across delete callers.

<!-- archie:ai-end -->
