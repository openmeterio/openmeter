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
- **As of:** the bound a caller asks about, meaning "usage before this
  instant". It is exclusive, like a period end and like a charge's service
  period end, so an as-of that is itself a period end belongs to the period
  ending there, not to the following one.
- **Realize after:** the earliest time a run may realize its period.

## The run as of a time

A run covers everything since the previous run's period ended, not only its own
period. The run as of a time is therefore the earliest run whose period ends
at or after that time:

- a time in a period that has a run gets that run;
- a time in a period that never got a run gets the next run that exists, and so
  does a time before the namespace's first run;
- a time after the namespace's latest run has no run yet, so the run for the
  first period end at or after it is created.

As-of is exclusive, and so is a period end, which decides what happens exactly
on a boundary. Charge runs often end at UTC midnight, and such a run holds only
usage of the day that just ended, so it must map to that day's run:

| As of | Run it belongs to |
| --- | --- |
| Nov 24, 10:00 | the run ending Nov 25, 00:00 |
| Nov 25, 00:00 exactly | the run ending Nov 25, 00:00 |
| Nov 25, 00:00:01 | the run ending Nov 26, 00:00 |

A run is never created behind an existing one. Usage of a skipped period is not
lost, because charge realizations are cumulative: the next run recognizes it.
Callers must read the period end from the run they get instead of deriving it
from the time they asked about.

A run is scheduled before its period ends, so that it exists by the time
anything needs to refer to it.

Usage up to a period end is only safe to read once the usage-based collection
delay has passed, so realize after is one collection delay past the period end.
It is stored on the run: changing the delay later does not move existing runs.

Each run points at the namespace's latest run at the time it was created, so
the runs of a namespace form a chain in period order. When periods were skipped,
the previous run is more than one period back.

Looking up or creating a run is serialized per namespace for the rest of the
caller's transaction, so two callers cannot chain to the same previous run.

## Intentional limitations

- Nothing starts or closes a run yet, so every run stays `scheduled`.
- The schedule is not configurable per namespace.
- Only usage-based charges take part. Flat fees and credit purchases are
  recognized differently.
