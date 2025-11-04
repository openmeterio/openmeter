# Examples

## Setup

Install dependencies

```sh
poetry install
```

## Running Examples

Run any example with environment variables:

```sh
OPENMETER_ENDPOINT=https://openmeter.cloud \
OPENMETER_TOKEN=om_xxx \
poetry run python ./sync/ingest.py
```

## Type Checking

The examples are type-checked using **Pyright**:

```sh
poetry run pyright
```

**Note**: Mypy is not compatible with the generated SDK models due to how it handles overloaded constructors. Pyright is the recommended type checker for this project.
