# driver

<!-- archie:ai-start -->

> Organisational layer housing the two concrete Watermill driver implementations: a full Kafka (IBM Sarama-backed) driver for production use and a noop driver for disabled/test paths. Code that needs to publish or subscribe to Watermill topics must choose one of these two children; no driver logic lives directly in this folder.

## Patterns

**Driver selection via Wire flag** — app/common selects kafka/ or noop/ at wiring time based on config flags (e.g. credits.enabled=false uses noop.Publisher). The driver folder itself contains no selection logic. (`Wire provider returns &noop.Publisher{} when feature is disabled; kafka.NewPublisher(...) otherwise.`)

## Anti-Patterns

- Adding source files directly in openmeter/watermill/driver/ — all driver code belongs in kafka/ or noop/ sub-packages
- Adding a third driver sub-package without a matching noop fallback for disabled-feature paths

## Decisions

- **Split into kafka/ and noop/ rather than a single driver with a mode flag** — Zero-value noop.Publisher{} is safe to use in disabled-feature wiring without any conditional branches at call sites; kafka/ keeps all Sarama/OTel complexity isolated.

<!-- archie:ai-end -->
