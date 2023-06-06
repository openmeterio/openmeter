# Event Format

To ensure seamless integration with infrastructure solutions and address common challenges like idempotency, we needed to choose an event format for usage ingestion.

## Context and Problem Statement

The event format needed to fulfill the following requirements:

- Integration with cloud infrastructure solutions.
- Support for multiple programming languages.
- Compatibility with various transport layers.
- Flexible payload definition.
- Support for batch ingestion.
- Uniqueness enforcement.
- Source and subject description.
- Uniqueness definition.

## Considered Options

1. [CloudEvents](https://cloudevents.io/)
2. Custom Format

## Decision Outcome

We have chosen CloudEvents.

### Consequences

- Pros: CloudEvents is a well-defined specification.
- Pros: CloudEvents enjoys strong support across various transports, languages, and libraries.
- Cons: CloudEvents may not provide sufficient specificity regarding payload requirements.
