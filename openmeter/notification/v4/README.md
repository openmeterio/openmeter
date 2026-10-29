# Notification v4

Package `v4` holds the Notification vNext model. It is built next to the
existing [Notifications](../README.md) domain, which keeps serving V1. This
README covers Event types and the Event Registry. Providers are documented in
[Svix](providers/svix/README.md) and [no-op](providers/noop/README.md).

Producing domains import `v4` for `EventType`, so `v4` must not import
Notification or any domain package; doing so would create an import cycle.

## Event types

An Event type is an unversioned name plus a positive version. Its canonical
string, `v<version>.<name>` such as `v1.billing.invoice.created`, is the
identity that crosses API, Kafka, persistence, and Provider boundaries.

- Names have at least two lowercase snake_case segments:
  `<resource>.<operation>` or `<subsystem>.<resource>.<operation>`. The same
  rule is enforced by the `@eventType` decorator at TypeSpec compile time; keep
  the two in sync.
- `ParseEventType` accepts only the canonical form: no sign or leading zeros in
  the version. Each type therefore has exactly one string, and parsing and
  `String` round-trip.
- Versions carry no compatibility guarantee. A released contract never changes;
  a changed contract is a new version next to the old one. Different versions
  are different types, and nothing converts Events or Rules between them.
- `EventType.Compare` orders by numeric version, then name. Sorting canonical
  strings would put `v10` before `v2`.

## Contracts belong to the producing domain

The domain that produces an Event type owns its payload and, for Rule-produced
types, its Rule configuration. Contracts are defined in TypeSpec and generated
into that domain's `notificationevents` package; see
[Notification Event contracts](../../../api/spec/README.md#notification-event-contracts-packagesaipsrcdomainnotification-events).
Notification validates payloads against the contract but never interprets
them.

For each contract, the generated package exposes the Event type as
`<Stem>V<n>EventType`, the embedded draft 2020-12 schemas as
`<Stem>V<n>PayloadJSONSchema` and, for Rule-produced types,
`<Stem>V<n>RuleConfigJSONSchema`, the matching Go structs, and the complete
`<Stem>V<n>Definition`. A schema's `$id` is `<event type>.payload` or
`<event type>.rule_config`.

`notificationcontracts.Definitions()` returns the definitions of every domain.
It is generated too, so a contract cannot be left out of the registry, and it
lives in the `notificationcontracts` subpackage because domain packages import
`v4`.

An `EventTypeDefinition` binds an Event type to its contracts:

- **Payload schema** is required.
- **Rule** is nil when no Rule can produce the type, as for system Events such
  as `v1.billing.invoice.created`. Otherwise it carries the Rule configuration
  schema.
- **DeprecatedBy** names the immediate replacement and is advisory. Deprecated
  types stay registered and usable; existing Events and Rules are never
  migrated. A type cannot replace itself, and following replacements must end
  at a type that is not deprecated.

`Validate` checks a definition on its own; references to other definitions are
checked by the registry. Callers with stricter needs, such as requiring a
Rule-capable type, add their own checks through `ValidateWith`.

Definitions are equal when their identity, schema bytes, Rule presence, and
replacement match. Schemas that differ only in formatting are not equal.

A definition is pure contract data. Domain validation of Rule configurations
needs domain services, such as checking that referenced features exist, so it
is not part of the definition: the owning domain provides the validator and
wiring passes it to the registry.

## Event Registry

`NewEventRegistry` builds the registry once, from every definition the process
serves. Nothing can be registered later, so the registry is immutable and safe
for concurrent use. Construction fails, and with it dependency injection, when:

- a definition is invalid, or two share an Event type;
- a schema does not compile;
- a schema references another document. Compilation loads no external
  documents, so a contract cannot change through a remote or local file it
  points to. Built-in metaschemas still resolve, and draft 2020-12 applies when
  a schema declares no `$schema`;
- `DeprecatedBy` names an unregistered type, or replacements form a cycle;
- a Rule configuration validator is nil, names a type that is unregistered or
  that Rules cannot produce, or names a type that already has one.

The registry stores deep copies of the definitions and returns copies, so a
caller cannot change a contract after it has been validated.

### Validation

- **Payloads** are checked against the schema only. Event creation is the only
  payload validation boundary: dispatch trusts the persisted canonical type and
  raw payload, so an older worker can dispatch a type that only a newer replica
  knows. The namespace argument is unused for now.
- **Rule configurations** are checked against the schema first, then by the
  domain validator, so domain validators only see well-formed configurations.
  This is the check for creating and updating a Rule.
- **Rule snapshots** on Events are checked against the schema only, through
  `RuleConfigSchemaValidator`. The domain already evaluated that
  configuration, and checking it against current domain state could reject a
  snapshot of a Rule that was valid when it ran.

Asking for either Rule validator of a type without `Rule` is an error.

Schema failures and Rule-less types return `models.GenericValidationError`.
Unregistered types return `models.GenericNotFoundError`. Domain validator
errors are returned unchanged, so the domain chooses their classification.

### Listing and patterns

`EventTypes` returns definitions in `EventType.Compare` order, optionally
filtered by an `EventTypePattern`. A pattern is either an exact canonical type
or ends in a `*` segment:

| Pattern | Matches |
|---|---|
| `v1.billing.invoice.created` | that type only |
| `v1.billing.*` | `v1.billing.invoice.created`, `v1.billing.invoice.updated`; not `v1.billingx.invoice.created` |
| `v1.*` | every version 1 type |

The version is always explicit, matching is case-sensitive, and the wildcard
matches at least one further segment. `*` alone, `v*`, and wildcards before the
last segment are rejected. A parsed pattern's `String` returns the same
canonical text, so a stored filter reads back unchanged. Parsing never consults
the registry, because a pattern may select types that only a newer release
registers. Channel Event type filters are meant to use the same grammar.

## Limitations

- The registry is process-local. Each replica knows only the types built into
  its binary, so during a rolling deployment replicas can disagree about which
  types exist.
- Released contracts are immutable by review rule only. No contract ledger or
  CI check detects an in-place change yet.
- Listing is not paginated; the list is small and held in memory.
