# hooks

<!-- archie:ai-start -->

> Structural folder owning entitlement lifecycle hooks that enforce cross-domain invariants on entitlement.Service operations. It has no direct source; its single child subscription/ guards subscription-owned entitlements against out-of-band mutation/deletion.

## Patterns

**ServiceHook over service-internal checks** — Invariants are enforced as entitlement ServiceHook implementations (embedding a Noop base) injected at wiring time rather than baked into entitlement.Service. (`subscription child embeds NoopServiceHook and overrides PreUpdate/PreDelete`)

## Anti-Patterns

- Encoding entitlement lifecycle invariants directly inside entitlement.Service instead of as a ServiceHook in this folder.
- Returning the concrete hook type from a constructor instead of the exported hook interface alias.

## Decisions

- **Cross-cutting entitlement ownership/lifecycle guards live as hooks under this folder, separate from the core service.** — Keeps entitlement.Service focused on CRUD while subscription-ownership enforcement is opt-in via DI and detectable from context, not a DB flag.

<!-- archie:ai-end -->
