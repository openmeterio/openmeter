# driver

<!-- archie:ai-start -->

> Organisational layer housing the two concrete Watermill driver implementations — kafka/ (IBM Sarama-backed, production) and noop/ (zero-value, disabled-feature paths). No driver logic lives directly here; all code belongs in the two sub-packages.

## Patterns

**Driver selection via Wire flag** — app/common selects kafka/ or noop/ at wiring time based on config flags. The driver folder contains no selection logic — the choice lives entirely in Wire provider functions in app/common. (`// app/common/watermill.go
if cfg.Credits.Enabled {
    return kafka.NewPublisher(brokerOpts)
}
return &noop.Publisher{}, nil`)
**No source files in driver/ root** — All driver code belongs exclusively in kafka/ or noop/ sub-packages. Adding .go files directly under openmeter/watermill/driver/ breaks the organisational contract. (`// BAD: openmeter/watermill/driver/mypublisher.go
// GOOD: openmeter/watermill/driver/kafka/publisher.go`)
**Each driver sub-package needs a noop counterpart** — Any new driver sub-package must have a matching noop fallback so disabled-feature Wire paths can substitute it without nil checks or conditional branches at call sites. (`// New driver added:
// openmeter/watermill/driver/redis/publisher.go
// Must be paired with:
// openmeter/watermill/driver/noop/publisher.go (already exists, satisfies interface)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `CLAUDE.md` | Archie-generated intent layer describing the organisational role of this folder and the two child sub-packages. | Do not delete the archie:ai-start/archie:ai-end markers — they are used by the Archie toolchain for drift detection and regeneration. |

## Anti-Patterns

- Adding .go source files directly under openmeter/watermill/driver/ — all code belongs in kafka/ or noop/ children
- Adding a third driver sub-package without a corresponding noop fallback for disabled-feature Wire paths
- Placing driver selection logic (config flag checks) inside this folder — selection belongs exclusively in app/common Wire provider functions
- Importing openmeter/watermill/driver directly from domain packages — domain code must go through the eventbus.Publisher interface, not the concrete driver

## Decisions

- **Split into kafka/ and noop/ sub-packages rather than a single driver with a runtime mode flag** — Zero-value noop.Publisher{} is safe to use in disabled-feature wiring without any conditional branches at call sites; kafka/ keeps all Sarama/OTel/SASL complexity isolated from the no-op path.
- **No code lives directly in the driver/ root folder** — Organisational folders with only child sub-packages make the architecture navigable — a developer opening driver/ immediately sees the two choices (kafka, noop) without scrolling through mixed source files.

<!-- archie:ai-end -->
