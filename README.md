# OpenMeter

[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/openmeterio/openmeter/ci.yaml?style=flat-square)](https://github.com/openmeterio/openmeter/actions/workflows/ci.yaml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/openmeterio/openmeter/badge?style=flat-square)](https://api.securityscorecards.dev/projects/github.com/openmeterio/openmeter)

<img src="assets/logo.png" width="100" alt="OpenMeter logo" />

----

OpenMeter is a Real-Time and Scalable Usage Metering for AI, Usage-Based Billing, Infrastructure, and IoT use-cases.

Learn more about OpenMeter on [https://openmeter.io](https://openmeter.io).

----

## Quickstart

Check out the [quickstart guide](/quickstart) for a 5-minute overview and demo of OpenMeter.

## Links

- [Examples](/examples)
- [Decisions](/docs/decisions)

## Examples

See our examples to learn about common OpenMeter use-cases.

- [Metering OpenAI Chat GPT Usage](/examples/ingest-openai-node)
- [Metering Kubernetes Pod Execution Time](/examples/ingest-kubernetes-pod-time-go)
- [Usage Based Billing with Stripe](/examples/export-stripe-go)

## Development

For the best developer experience, install [Nix](https://nixos.org/download.html) and [direnv](https://direnv.net/docs/installation.html).

Run the dependencies:

```sh
make up
```

Run OpenMeter:

```sh
make run
```

## Roadmap

Visit our website at [https://openmeter.io](https://openmeter.io#roadmap) for our public roadmap.

## License

The project is licensed under the [Apache 2.0 License](LICENSE).

[![FOSSA Status](https://app.fossa.com/api/projects/custom%2B38090%2Fgithub.com%2Fopenmeterio%2Fopenmeter.svg?type=large)](https://app.fossa.com/projects/custom%2B38090%2Fgithub.com%2Fopenmeterio%2Fopenmeter?ref=badge_large)
