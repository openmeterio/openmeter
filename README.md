<p align="center">
  <a href="https://openmeter.io">
    <img src="assets/logo.png" width="100" alt="OpenMeter logo" />
  </a>

  <h1 align="center">
    OpenMeter
  </h1>
</p>

[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/openmeterio/openmeter/ci.yaml?style=flat-square)](https://github.com/openmeterio/openmeter/actions/workflows/ci.yaml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/openmeterio/openmeter/badge?style=flat-square)](https://api.securityscorecards.dev/projects/github.com/openmeterio/openmeter)

OpenMeter is a Real-Time and Scalable Usage Metering for AI, Usage-Based Billing, Infrastructure, and IoT use-cases.

Learn more about OpenMeter at [https://openmeter.io](https://openmeter.io).

## Quickstart

Check out the [quickstart guide](/quickstart) for a 5-minute overview and demo of OpenMeter.

## Links

- [Examples](/examples)
- [Demo Video](https://www.loom.com/share/c965e56f1df9450492e687dfb3c18b49)
- [Stripe UBP Demo](https://www.loom.com/share/bc1cfa1b7ed94e65bd3a82f9f0334d04)
- [Decisions](/docs/decisions)

## Community

To engage with our community, you can use the following resources:

- [Discord](https://discord.gg/nYH3ZQ3Xzq) - Get support or discuss the project.
- [Contributing to OpenMeter](CONTRIBUTING.md) - Start here if you want to contribute.
- [Code of Conduct](CODE_OF_CONDUCT.md) - Our community guidelines.
- [Adopters](ADOPTERS.md) - Companies already using OpenMeter.
- [Blog](https://openmeter.io/blog/) - Stay up-to-date.

## Examples

See our examples to learn about common OpenMeter use-cases.

- [Metering OpenAI Chat GPT Usage](/examples/ingest-openai-node)
- [Metering Kubernetes Pod Execution Time](/examples/ingest-kubernetes-pod-time-go)
- [Usage Based Billing with Stripe](/examples/export-stripe-go)

## API

OpenMeter exposes a [REST API](https://editor.swagger.io/?url=https://raw.githubusercontent.com/openmeterio/openmeter/main/api/openapi.yaml) for integrations.

## Client SDKs

Currently, we offer the following Client SDKs:

- [Node.js](/api/client/node)
- [Python](/api/client/python)
- [Go](/api/client/go)
- [Web](/api/client/web)

In cases where no specific SDK is available for your preferred programming language, you can utilize the [OpenAPI definition](https://github.com/openmeterio/openmeter/blob/main/api/openapi.yaml).
Please raise a [GitHub issue](https://github.com/openmeterio/openmeter/issues/new?assignees=&labels=area%2Fapi%2Ckind%2Ffeature&projects=&template=feature_request.yaml) to request SDK support in other languages.

## Development

**For an optimal developer experience, it is recommended to install [Nix](https://nixos.org/download.html) and [direnv](https://direnv.net/docs/installation.html).**

<details><summary><i>Installing Nix and direnv</i></summary><br>

**Note: These are instructions that _SHOULD_ work in most cases. Consult the links above for the official instructions for your OS.**

Install Nix:

```sh
sh <(curl -L https://nixos.org/nix/install) --daemon
```

Consult the [installation instructions](https://direnv.net/docs/installation.html) to install direnv using your package manager.

On MacOS:

```sh
brew install direnv
```

Install from binary builds:

```sh
curl -sfL https://direnv.net/install.sh | bash
```

The last step is to configure your shell to use direnv. For example for bash, add the following lines at the end of your `~/.bashrc`:

    eval "\$(direnv hook bash)"

**Then restart the shell.**

For other shells, see [https://direnv.net/docs/hook.html](https://direnv.net/docs/hook.html).

**MacOS specific instructions**

Nix may stop working after a MacOS upgrade. If it does, follow [these instructions](https://github.com/NixOS/nix/issues/3616#issuecomment-662858874).

<hr>
</details>

Run the dependencies:

```sh
make up
```

Run OpenMeter Sink Worker

```sh
make sink-worker
```

Run OpenMeter:

```sh
make server
```

Run tests:

```sh
make test
```

Run linters:

```sh
make lint
```

## Roadmap

Visit our website at [https://openmeter.io](https://openmeter.io#roadmap) for our public roadmap.

## License

The project is licensed under the [Apache 2.0 License](LICENSE).

[![FOSSA Status](https://app.fossa.com/api/projects/custom%2B38090%2Fgithub.com%2Fopenmeterio%2Fopenmeter.svg?type=large)](https://app.fossa.com/projects/custom%2B38090%2Fgithub.com%2Fopenmeterio%2Fopenmeter?ref=badge_large)
