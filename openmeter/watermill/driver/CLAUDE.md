# driver

<!-- archie:ai-start -->

> Structural folder owning the Watermill message.Publisher/Subscriber transport implementations. It splits by concrete transport: kafka/ is the real Sarama-backed production driver; noop/ is the null publisher that silently discards messages when event publishing is disabled. Both feed openmeter/watermill/eventbus, which selects between them.

## Patterns

**Transport selection lives above this folder** — Driver subpackages only construct a Publisher/Subscriber; the choice of kafka vs noop is made by the consumer (eventbus / app wiring), not inside a driver. Keep drivers ignorant of when they are chosen. (`eventbus uses noop.Publisher{} when publishing is disabled, else the kafka publisher from broker.go`)
**Every driver satisfies the same Watermill contract via compile-time assertion** — Each concrete driver must assert the Watermill interface it implements so a transport swap stays type-safe. (`noop/publisher.go: var _ message.Publisher = (*Publisher)(nil)`)
**Config-driven construction with explicit dependency injection** — The real driver (kafka) takes an Options/BrokerOptions struct validated via Validate() before building Sarama/Watermill objects; loggers and meters are injected, never defaulted via slog.Default(). (`kafka/broker.go: in.Validate() then createKafkaConfig(role) before publisher/subscriber creation`)

## Anti-Patterns

- Adding a new transport that skips the message.Publisher/Subscriber compile-time interface assertion — breaks the swappability contract eventbus relies on.
- Pushing transport-selection logic (kafka-vs-noop) down into a driver subpackage instead of keeping it in eventbus/app wiring.
- Leaking Sarama/Kafka config or partition-key logic into the noop driver — noop must stay dependency-free and side-effect-free.

## Decisions

- **Drivers are split by concrete transport (kafka, noop) as sibling subpackages rather than one package with a mode flag.** — Lets noop stay zero-dependency and trivially value-receiver based, while kafka pulls in Sarama, SASL/SCRAM, and OTel metrics without leaking those concerns into the disabled path.
- **The null/disabled publisher is a real driver (noop) rather than a nil publisher or per-call-site guards.** — Consumers always hold a valid message.Publisher, so publish call sites need no nil checks regardless of whether eventing is enabled.

<!-- archie:ai-end -->
