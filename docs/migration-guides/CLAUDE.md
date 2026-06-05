# migration-guides

<!-- archie:ai-start -->

> Dated, version-anchored migration guides for breaking or deprecating changes operators must act on when upgrading OpenMeter. Documentation only — gives manual steps, SQL, and SDK before/after diffs; no executable code lives here.

## Patterns

**Date-prefixed filenames** — Each guide is `YYYY-MM-DD-topic.md`, ordering guides chronologically by when the change was introduced. (`2025-11-04-entitlement-events-v1.md is the most recent`)
**Version range header for upgrade-path guides** — Guides tied to a release pin source/target versions at the top so operators know exactly which upgrade triggers the steps. (`2025-06-26-subscription-alignment.md opens with 'From Version: v1.0.0-beta.214 / To Version: v1.0.0-beta.215'`)
**Repo-relative source links to ground claims** — Behavioral assertions link to the exact Go file+line implementing them so the doc stays verifiable against code. (`subscription-alignment links to /openmeter/productcatalog/alignment.go#L22 and errors.go#L411`)
**Before/After code diffs plus copy-paste SQL** — Guides show TS SDK before/after snippets for API changes and runnable Postgres SQL (SELECT-count-then-UPDATE) for data migrations. (`subject-customer-consolidation shows openmeter.subjects.createEntitlement → openmeter.customers.createEntitlement`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `2025-06-26-subscription-alignment.md` | Migrates unaligned subscriptions/plans to aligned; gives SQL to find convertible rows (every RateCard cadence matching plan billing_cadence) and bulk-UPDATE billables_must_align=true, flagging the rest for manual resolution. | Operates directly on `plans`, `plan_phases`, `plan_rate_cards`, `subscriptions`, `subscription_phases`, `subscription_items` tables; BillablesMustAlign config is slated for removal in beta.216. |
| `2025-08-12-subject-customer-consolidation.md` | Documents subjects→customers consolidation: customers become the primary entity, usage auto-attributed by customer ID/Key, /subjects APIs deprecated then removed, multiple subjects per customer via usageAttribution. | Subject values colliding with an existing customer.key/customer.id silently misattribute usage; deprecation timeline (deprecated Sep 1 2025, removed Nov 1 2025). |
| `2025-11-04-entitlement-events-v1.md` | Announces removal of V1 Entitlement Events; V2 introduced in commit 85f7ec90 due to breaking schema changes. | Operators MUST drain all in-flight V1 events before upgrading, especially when coming from a version prior to 85f7ec90. |

## Anti-Patterns

- Describing a breaking change in prose without the pinned version range / commit anchor an operator needs to act
- Providing destructive SQL without a preceding SELECT/COUNT verification step (every guide here counts first)
- Linking behavior claims to no code reference, making the guide impossible to validate against the implementation

## Decisions

- **Limit alignment migration SQL to the common 'RateCard cadence == plan cadence' case** — subscription-alignment notes a fully exhaustive PLPGSQL check is possible but most unaligned plans share literal cadence values, so simpler SQL covers the bulk and flags the rest for manual review.

<!-- archie:ai-end -->
