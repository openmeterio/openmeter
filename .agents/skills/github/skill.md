# GitHub API Skill

Use `gh api` for GitHub operations instead of high-level `gh` subcommands when they fail due to deprecated features.

## Why

The `gh pr edit`, `gh pr create`, and similar commands internally query GitHub's Projects (classic) API, which is deprecated. This causes failures like:

```
GraphQL: Projects (classic) is being deprecated in favor of the new Projects experience
```

These commands exit with code 1 even though the actual operation (e.g., updating the PR body) would succeed — the error comes from a secondary GraphQL query for classic projects.

## Workarounds

### Update PR description

Instead of:
```bash
gh pr edit 3986 --body-file /tmp/body.md  # FAILS with Projects classic error
```

Use the REST API directly:
```bash
gh api repos/openmeterio/openmeter/pulls/3986 -X PATCH -F "body=@/tmp/body.md" --jq '.html_url'
```

### Create PR

Instead of `gh pr create`, use:
```bash
gh api repos/openmeterio/openmeter/pulls -X POST \
  -f title="PR title" \
  -f head="branch-name" \
  -f base="main" \
  -F "body=@/tmp/body.md" \
  --jq '.html_url'
```

### Read PR comments

```bash
# Review comments (inline code comments)
gh api repos/openmeterio/openmeter/pulls/3986/comments --jq '.[] | {id, path, line, body, user: .user.login}'

# Issue-level comments
gh pr view 3986 --comments --json comments --jq '.comments[] | {author: .author.login, body: .body}'

# Reviews
gh api repos/openmeterio/openmeter/pulls/3986/reviews --jq '.[] | {id, user: .user.login, state, body}'
```

### General pattern

For any `gh` subcommand that fails with the Projects classic deprecation error, rewrite using `gh api` with the corresponding REST endpoint. The `-F` flag reads file contents with `@path`, `-f` sends string fields.
