# Periodic realization

Periodic realization gives usage-based revenue a regular recognition point. At
the end of every period, usage-based charges that are in the middle of their
service period get a checkpoint realization, so usage is recognized up to the
period end instead of only when the charge is invoiced or ends.

This package owns the namespace-level side of that: which periods exist and the
run that belongs to each period. It does not rate usage or create charge
realizations; those belong to
[Charges](../charges/README.md#realization-and-time-semantics).

## Vocabulary

- **Period:** the window a run realizes usage in. It includes its start and
  excludes its end. Periods follow one schedule shared by every namespace: UTC
  days.
- **Period to:** the end of a period. It identifies the run: a namespace has at
  most one run per period end.
- **Run:** a namespace's realization run for one period.
- **As of:** the instant a caller asks about. The run as of a time is the run of
  the period that contains it, so a time that is itself a period end belongs to
  the following period.
- **Realize after:** the earliest time a run may realize its period.

## Scheduling a run

A run is scheduled before its period ends, so that it exists by the time
anything needs to refer to it. Scheduling as of a time creates the run of the
period that contains that time.

Usage up to a period end is only safe to read once the usage-based collection
delay has passed, so realize after is one collection delay past the period end.
It is stored on the run: changing the delay later does not move existing runs.

Scheduling is serialized per namespace for the rest of the caller's transaction.

## Intentional limitations

- Only the first run of a namespace can be scheduled. Scheduling into a
  namespace that already has a run is rejected as not implemented.
- Nothing starts or closes a run yet, so every run stays `scheduled`.
- The schedule is not configurable per namespace.
- Only usage-based charges take part. Flat fees and credit purchases are
  recognized differently.
