# OpenMeter

## Tips for working with the codebase

Development commands are run via `Makefile`, it contains all commonly used commands during development. `Dagger` and `justfile` are also present but seldom used. Use the Makefile commands for common tasks like running tests, generating code, linting, etc.

## Testing

To run all tests, invoke `make test` or `make test-nocache` if you want to bypass the test cache.

When running tests for a single file or testcase (invoking directly and not with Make), make sure the environment is set correctly. Examples of a correct setup can be found in the `Makefile`'s `test` command, or in `.vscode/settings.json` `go.testEnvVars`. Example command would be:

E2E tests are run via `make etoe`, they are API tests that need to start dependencies via docker compose, always invoke them via Make.

## Code Generation

Some directories are generated from code, never edit them manually. A non-exhaustive list of them is:
- `make gen-api`: generates from TypeSpec
  - the clients in `api/client`
  - the OAPI spec in `api/openapi.yaml`
- `make generate`: runs go codegen steps
  - database access in `**/ent/db` from the ent schema in `**/ent/schema`
  - dependency injection with wire in `**/wire_gen.go` from `**/wire.go`
- `atlas migrate --env local diff <migration-name>`: generates a migration diff from changes in the generated ent schema (in `tools/migrate/migrations`)

