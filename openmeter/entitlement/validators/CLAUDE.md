# validators

<!-- archie:ai-start -->

> Structural folder holding validators that hook the entitlement domain into other domains' request lifecycles. It has no direct source; its child customer/ implements customer.RequestValidator to block deletion of customers that still have active entitlements.

## Patterns

**Cross-domain RequestValidator implementation** — Validators implement another domain's RequestValidator contract (e.g. customer.RequestValidator), embed that domain's Noop base, and depend on entitlement.EntitlementRepo (not the full Service) for read-only checks. (`customer child embeds customer.NoopRequestValidator and queries EntitlementRepo.ListEntitlements at clock.Now()`)

## Anti-Patterns

- Placing entitlement-aware validation logic in the customer domain instead of here, inverting the dependency direction.
- Injecting entitlement.Service when only EntitlementRepo (list access) is required.

## Decisions

- **Validators that depend on entitlement state live in the entitlement domain and implement the consuming domain's validator interface.** — The entitlement domain may import customer, but not vice versa; placing the validator here keeps the dependency acyclic while still letting customer deletes reject when active entitlements exist.

<!-- archie:ai-end -->
