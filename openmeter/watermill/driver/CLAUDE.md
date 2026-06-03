# driver

<!-- archie:ai-start -->

> Organisational layer housing the two concrete Watermill driver implementations — kafka/ (IBM Sarama-backed, production) and noop/ (zero-value message.Publisher for disabled-feature Wire paths). No driver logic lives directly in this root; all code belongs in the two sub-packages, and driver selection happens only in app/common Wire providers.

## Patterns

**Driver selection via Wire flag** — app/common picks kafka/ or noop/ at wiring time based on config flags. The driver folder contains no selection logic — the choice lives entirely in Wire provider functions in app/common. (`// app/common/watermill.go
if cfg.Credits.Enabled { return kafka.NewPublisher(brokerOpts) }
return &noop.Publisher{}, nil`)
**No source files in driver/ root** — All driver code belongs exclusively in kafka/ or noop/. Adding .go files directly under openmeter/watermill/driver/ breaks the organisational contract. (`// BAD: openmeter/watermill/driver/mypublisher.go  GOOD: openmeter/watermill/driver/kafka/publisher.go`)
**Each driver sub-package needs a noop counterpart** — Any new driver sub-package must have a matching noop fallback so disabled-feature Wire paths can substitute it without nil checks or conditional branches at call sites. (`// new driver -> driver/redis/publisher.go must be paired with a noop.Publisher satisfying the interface`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `CLAUDE.md` | Archie-generated intent layer describing the organisational role of this folder and its two child sub-packages (kafka, noop). | Do not delete the archie:ai-start/archie:ai-end markers — the Archie toolchain uses them for drift detection and regeneration. |

## Anti-Patterns

- Adding .go source files directly under openmeter/watermill/driver/ — all code belongs in kafka/ or noop/ children
- Adding a third driver sub-package without a corresponding noop fallback for disabled-feature Wire paths
- Placing driver selection logic (config flag checks) inside this folder — selection belongs exclusively in app/common Wire provider functions
- Importing openmeter/watermill/driver directly from domain packages — domain code must go through the eventbus.Publisher interface, not the concrete driver

## Decisions

- **Split into kafka/ and noop/ sub-packages rather than one driver with a runtime mode flag** — Zero-value noop.Publisher{} is safe in disabled-feature wiring without conditional branches at call sites; kafka/ keeps all Sarama/OTel/SASL complexity isolated from the no-op path.
- **No code lives directly in the driver/ root folder** — An organisational folder with only child sub-packages makes the two choices (kafka, noop) immediately visible without scrolling through mixed source files.

## Example: Select the Kafka or noop publisher in a Wire provider gated by a feature flag

```
import (
  "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
  "github.com/openmeterio/openmeter/openmeter/watermill/driver/noop"
)
func ProvidePublisher(cfg config.Configuration, opts kafka.BrokerOptions) (message.Publisher, error) {
  if !cfg.Events.Enabled { return &noop.Publisher{}, nil }
  return kafka.NewPublisher(opts)
}
```

<!-- archie:ai-end -->
