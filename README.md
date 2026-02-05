<p align="center">
  <a href="https://openmeter.io">
    <img src="assets/logo.png" width="100" alt="OpenMeter logo" />
  </a>

  <h1 align="center">
    OpenMeter
  </h1>
</p>

[![GitHub Release](https://img.shields.io/github/v/release/openmeterio/openmeter?style=flat-square)](https://github.com/openmeterio/openmeter/releases/latest)
[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/openmeterio/openmeter/ci.yaml?style=flat-square)](https://github.com/openmeterio/openmeter/actions/workflows/ci.yaml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/openmeterio/openmeter/badge?style=flat-square)](https://api.securityscorecards.dev/projects/github.com/openmeterio/openmeter)
[![Go Report Card](https://goreportcard.com/badge/github.com/openmeterio/openmeter?style=flat-square)](https://goreportcard.com/report/github.com/openmeterio/openmeter)
![GitHub Repo stars](https://img.shields.io/github/stars/openmeterio/openmeter)
![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/openmeterio?label=Follow)

OpenMeter provides flexible Billing and Metering for AI and DevTool companies. It also includes real-time insights and usage limit enforcement.

Learn more about OpenMeter at [https://openmeter.io](https://openmeter.io).

## Try It

Get started with the latest version of OpenMeter in minutes.

### Local

```sh
git clone git@github.com:openmeterio/openmeter.git
cd openmeter/quickstart
docker compose up -d
```

Check out the [quickstart guide](/quickstart) for a 5-minute overview and demo of OpenMeter.

### Cloud

[Sign up](https://openmeter.cloud) for a free account and start metering your usage in the cloud.

> [!TIP]
> Check out how OpenMeter Cloud compares with the self-hosted version in our [comparison guide](https://openmeter.io/docs/cloud#comparison).

### Deploy

Deploy OpenMeter to your Kubernetes cluster using our [Helm chart](https://openmeter.io/docs/deploy/kubernetes).

## Links

- [Examples](/examples)
- [Demo Video](https://www.loom.com/share/c965e56f1df9450492e687dfb3c18b49)
- [Stripe UBP Demo](https://www.loom.com/share/bc1cfa1b7ed94e65bd3a82f9f0334d04)
- [Decisions](/docs/decisions)
- [Migration Guides](/docs/migration-guides)

## Community

To engage with our community, you can use the following resources:

- [Discord](https://discord.gg/nYH3ZQ3Xzq) - Get support or discuss the project.
- [Contributing to OpenMeter](CONTRIBUTING.md) - Start here if you want to contribute.
- [Code of Conduct](CODE_OF_CONDUCT.md) - Our community guidelines.
- [Adopters](ADOPTERS.md) - Companies already using OpenMeter.
- [Blog](https://openmeter.io/blog/) - Stay up-to-date.

## Examples

See our examples to learn about common OpenMeter use-cases.

- [Metering Kubernetes Pod Execution Time](/examples/collectors/kubernetes-pod-exec-time)
- Usage Based Billing with Stripe ([Go](https://github.com/openmeterio/examples/tree/main/export-stripe-go), [Node](https://github.com/openmeterio/examples/tree/main/export-stripe-node))
- [Metering based on logs](/examples/ingest-logs)

## API

OpenMeter exposes a [REST API](https://editor.swagger.io/?url=https://raw.githubusercontent.com/openmeterio/openmeter/main/api/openapi.yaml) for integrations.

## Client SDKs

Currently, we offer the following Client SDKs:

- [JavaScript](/api/client/javascript)
- [Python](/api/client/python)
- [Go](/api/client/go)

In cases where no specific SDK is available for your preferred programming language, you can utilize the [OpenAPI definition](https://github.com/openmeterio/openmeter/blob/main/api/openapi.yaml).
Please raise a [GitHub issue](https://github.com/openmeterio/openmeter/issues/new?assignees=&labels=area%2Fapi%2Ckind%2Ffeature&projects=&template=feature_request.yaml) to request SDK support in other languages.

## Key Concepts

### Multi-Tenancy
- **Namespace** - Logical isolation boundary that segments data for multi-tenancy

### Customers & Subjects
- **Customer** - An entity representing a client who uses the service
- **Subject** - An individual user or entity within a customer account, used for fine-grained usage tracking

### Metering & Events
- **Event** - A recorded occurrence of usage activity (e.g., API call, feature usage) with timestamp and metadata
- **Meter** - Defines how to aggregate events (e.g., sum of API calls, unique users)

### Product Catalog
- **Feature** - A capability that can be metered and entitled
- **Plan** - A versioned bundle of features with pricing, organized into phases
- **Plan Phase** - A time-bounded segment within a plan with specific pricing and rate cards
- **Addon** - Optional supplementary features that can be attached to plans or subscriptions
- **Rate Card** - Pricing specification for a feature, defining price type, billing cadence, and discounts

### Entitlements & Credits
- **Entitlement** - Grants a customer access to a feature with optional limits and usage tracking
- **Grant** - A credit allocation to an entitlement with amount, priority, and expiration

### Subscriptions
- **Subscription** - Links a customer to a plan with billing lifecycle and cadence
- **Subscription Phase** - A time-bounded segment of subscription execution
- **Subscription Item** - A billable product item within a subscription phase

### Billing
- **Billing Profile** - Configuration defining billing provider setup, tax settings, and supplier details
- **Invoice** - A billable document generated from subscription items and usage charges
- **Invoice Line** - A line item representing a charge (flat fee or usage-based) with quantity and pricing

### Apps & Integrations
- **App** - An installed integration/provider (e.g., Stripe) for payments, invoicing, or tax calculation

## Development

### Prerequisites

**Recommended:** Install [Nix](https://nixos.org/download.html) and [direnv](https://direnv.net/docs/installation.html) - the project's `flake.nix` provides all required tools automatically.

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

**Manual setup:** Go 1.23+, Docker, librdkafka, Node.js (for TypeSpec), Atlas CLI (for migrations).

### Project Layout

```
api/                  # API definitions and generated code
  spec/src/           # TypeSpec source files (API-first design)
  client/             # Generated SDKs (Go, JS, Python)
  v3/                 # v3 API handlers and generated code
app/                  # Application wiring (Wire providers)
cmd/                  # Service entry points
openmeter/            # Core business logic packages
pkg/                  # Shared utilities
tools/migrate/        # Database migrations
e2e/                  # End-to-end tests
config.example.yaml   # Configuration reference
```

### Quick Start

```sh
# Start dependencies (PostgreSQL, Kafka, ClickHouse)
make up

# Run the API server with hot-reload
make server

# In another terminal, run tests
make test
```

Configuration is loaded from `config.yaml` (copy `config.example.yaml` to get started).

### Docker Compose Profiles

By default, `make up` starts core dependencies. Use `COMPOSE_PROFILES` to enable optional services:

```sh
# Start all services including optional ones
COMPOSE_PROFILES=dev,redis,webhook make up
```

Available profiles:
- `dev` - ClickHouse UI and Kafka UI for debugging
- `redis` - Redis for caching/deduplication
- `webhook` - Svix for webhook delivery

### Services

OpenMeter consists of multiple services that can be run independently:

```sh
make server              # API server (main entry point)
make sink-worker         # Processes events from Kafka → ClickHouse
make balance-worker      # Manages entitlement balances
make billing-worker      # Processes billing operations
make notification-service # Handles webhook notifications
```

All services support hot-reload via [air](https://github.com/air-verse/air).

### Common Commands

```sh
# Building
make build               # Build all binaries
make build-server        # Build specific binary

# Testing
make test                # Run tests
make test-nocache        # Run tests bypassing cache
make etoe                # Run e2e tests (starts own dependencies)

# Linting
make lint                # Run all linters
make lint-go             # Run Go linter only
make fmt                 # Auto-fix lint issues

# Code Generation
make generate            # Regenerate ent schemas + Wire DI
make gen-api             # Regenerate TypeSpec → OpenAPI → clients
```

### Testing

Tests require PostgreSQL. When running directly (not via Make):

```sh
docker compose up -d postgres
TZ=UTC POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -run TestName ./path/to/package
```

The `-tags=dynamic` flag is required to build against local librdkafka.

### Code Generation

OpenMeter uses code generation extensively. Never edit these files manually:

- `api/openapi.yaml`, `api/v3/openapi.yaml` - Generated from TypeSpec
- `api/api.gen.go`, `api/v3/api.gen.go` - Generated from OpenAPI
- `openmeter/ent/db/` - Generated from ent schemas
- `cmd/*/wire_gen.go` - Generated by Wire

### Database Migrations

Uses [Atlas](https://atlasgo.io/) with ent schema as source:

```sh
# Generate a new migration after changing ent schema
atlas migrate --env local diff <migration-name>
```

Migrations are stored in `tools/migrate/migrations/`.

### Troubleshooting

If you see `ghcr.io denied` errors:

```sh
docker login ghcr.io
```

Use a GitHub personal access token with `read:packages` scope.

## Roadmap

Visit our website at [https://openmeter.io](https://openmeter.io#roadmap) for our public roadmap.

## License

The project is licensed under the [Apache 2.0 License](LICENSE).

[![FOSSA Status](https://app.fossa.com/api/projects/custom%2B38090%2Fgithub.com%2Fopenmeterio%2Fopenmeter.svg?type=large)](https://app.fossa.com/projects/custom%2B38090%2Fgithub.com%2Fopenmeterio%2Fopenmeter?ref=badge_large)
