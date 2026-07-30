# Subscription patches

Patches are typed, ordered transformations of a `SubscriptionSpec`. They let
subscription workflows express a change to desired state before the core
service materializes that state.

Patches do not persist subscription rows, schedule entitlements, or create
billing artifacts.

## Patch contract

Every patch provides:

- an operation such as add, remove, stretch, or unschedule
- a `SpecPath` identifying the affected phase, item key, or item version
- validation of its own input
- an `ApplyTo` transformation over the spec

Paths express logical subscription identity:

```text
/phases/{phaseKey}
/phases/{phaseKey}/items/{itemKey}
/phases/{phaseKey}/items/{itemKey}/idx/{version}
```

Use an item-version path only when the operation addresses that exact version.
An operation that selects or mutates the relevant version itself belongs at the
item path.

## Effective time and ordering

`ApplyContext.CurrentTime` is the command's resolved effective time, not
necessarily wall-clock time. Patch rules use it to protect past phases and
items and to decide which item version is active.

Patches are applied in order. A later patch sees the spec produced by earlier
ones, so valid workflows may deliberately remove an item before adding its
replacement. The complete spec is validated after application.

Adding an item to the current phase closes the active version of the same item
key and appends a new version. Its relative start is derived from the effective
time when the patch does not provide one. Future-phase additions must not
silently replace an existing item.

Phase stretch and removal can shift later phase offsets. They must preserve
phase ordering and cannot erase a phase by collapsing its duration.

## Error meaning

- validation errors mean the patch or resulting spec is structurally invalid
- forbidden errors mean the requested edit would change protected historical
  state
- conflict errors mean the patch is individually meaningful but incompatible
  with the surrounding spec or another requested change

The workflow layer maps these errors for callers and passes the resulting
complete spec to the [subscription service](../service/README.md).
