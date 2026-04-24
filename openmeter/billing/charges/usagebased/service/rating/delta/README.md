# Delta Rating

Delta rating is the temporary production detailed-line engine for usage-based
charges. It rates the latest cumulative meter snapshot, subtracts every detailed
line already booked for the charge, and books the remaining delta on the current
run's service period.

The engine intentionally does not preserve the original service period of
corrections. That keeps downstream invoicing in the simpler "current invoice
period only" shape while the period-preserving engine and invoicing support for
corrections evolve.

Repricing is expected with non-linear prices. For example, a volume-tiered price
can rate `15` units at one unit amount and `16` units at a different unit
amount for the whole quantity. Delta rating handles this by emitting a reversal
for the previously booked price component and a current line for the newly rated
price component, both on the current run's service period.

## Algorithm

Input:

- the usage-based charge intent
- the current run service period and cumulative metered quantity
- all detailed lines already billed by prior eligible realization runs

Calculation:

1. Generate billing-rating detailed lines for the current cumulative snapshot.
   The snapshot covers `[intent.ServicePeriod.From, currentPeriod.To)`.
2. Ignore minimum commitment while the current period is not the final charge
   service-period snapshot.
3. Convert billing-rating detailed lines to usage-based detailed lines.
   `PricerReferenceID` stores the billing-rating child reference used for
   arithmetic matching. `ChildUniqueReferenceID` starts from the generated child
   reference, but generated billing-rating output can contain duplicates and the
   subtraction generator can rewrite it for correction lines.
4. Remove credit allocations from already billed lines before arithmetic.
   Credits are allocated after rating, so credit changes must not look like
   usage or pricing changes.
5. Call `subtract.SubtractRatedRunDetails(current, alreadyBilled, generator)`.
6. Stamp every remaining line to the current run service period and clear
   `CorrectsRunID`.
7. Sort output lines, assign dense indexes, and validate that final output
   `ChildUniqueReferenceID` values are unique.

The result totals are the sum of the emitted detailed lines.

## Example: Additional Unit Usage

Price: unit price `10`.

Already billed in period 1:

```text
5 units @ 10 = 50
```

Current cumulative snapshot at period 2: `8` units.

Billing rating generates the cumulative state:

```text
8 units @ 10 = 80
```

Delta subtracts what was already booked:

```text
8 units @ 10 - 5 units @ 10 = +3 units @ 10 = 30
```

Output:

```text
3 units @ 10 = 30, service period = period 2
```

## Example: Volume-Tiered Repricing Correction

Price: volume-tiered price where `1..15` units rate at `$10/unit`, and `16+`
units rate at `$5/unit` for the whole quantity.

Already billed:

```text
previous := 15 units @ $10 = $150, detailed line id = phase-1-line-1
```

Current cumulative rating:

```text
current := 16 units @ $5 = $80
```

Delta output:

```text
previous-only reversal := -15 units @ $10 = -$150, child ref = volume-tiered-price#correction:detailed_line_id=phase-1-line-1
current-only           := +16 units @ $5  = +$80,  child ref = volume-tiered-price
```

The two lines share the same `PricerReferenceID`, but their `PerUnitAmount`
differs. Subtraction therefore treats this as repricing instead of a quantity
delta. Both output lines are stamped to the current run service period. The
correction child reference is deterministic and points at the previous detailed
line ID so persistence can distinguish the reversal from the current generated
line.
