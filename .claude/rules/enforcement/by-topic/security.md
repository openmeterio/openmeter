# Enforcement: security (1 rule)

Topic file. Loaded on demand when an agent works on something in the `security` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pattern Divergence (inform)

### `infra-secrets-001` — Never commit real secrets; Trufflehog secret scanning fails the build on findings

*source: `deep_scan`*

**Why:** Trufflehog secret scanning runs on PRs/pushes to main and fails on findings. Release secrets are provided via GitHub Actions (GITHUB_TOKEN, POETRY_PYPI_TOKEN_PYPI, vars.DEPOT_PROJECT, vars.TEST_CLICKHOUSE_DSN); SVIX_JWT_SECRET=DUMMY_JWT_SECRET is a non-sensitive dev value only. npm publishing uses OIDC Trusted Publishing (no token).
