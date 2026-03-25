---
name: rebase
description: Rebase an OpenMeter branch onto another branch and handle repo-specific gotchas like sequential migrations, atlas.sum conflicts, Ent regeneration, and targeted verification.
user-invocable: true
argument-hint: "[target branch or rebase situation]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Rebase Gotchas

Use this skill when rebasing an OpenMeter branch, especially onto `origin/main`, and there may be migration, Ent, or generated-code conflicts.

## Before rebasing

- Check `git status --short --branch` first.
- Do not start a rebase on top of unrelated uncommitted work without preserving it first.
- Fetch the target branch explicitly before rebasing.

## Standard flow

1. `git fetch origin main`
2. `git rebase origin/main`
3. Resolve conflicts one commit at a time.
4. Prefer `GIT_EDITOR=true git rebase --continue` in non-interactive shells.

## First rule for conflicts

When a rebase conflict appears, the first move should usually be regenerating generated artifacts:

```bash
make generate
```

Treat this as the default first step before manually resolving conflicted generated files.

Why:

- conflicts often come from generated Ent code
- conflicts can come from API-generated code
- migration-related schema changes often need fresh generated output too

In practice, regenerating first resolves or clarifies most conflicts much faster than hand-merging generated files.

## Migration conflicts

OpenMeter migrations must stay sequential by timestamp.

If a rebased commit introduces a migration that is now older than migrations already on the target branch:

- Remove the older migration pair instead of keeping both.
- Do not hand-edit the SQL to force it in.
- Recreate the migration at the new head after regeneration.

Typical flow:

1. Remove the outdated `.up.sql` and `.down.sql` files.
2. Regenerate code:

```bash
make generate
```

3. Generate a fresh migration at the current head:

```bash
atlas migrate --env local diff <migration-name>
```

4. Stage the newly generated migration pair and `tools/migrate/migrations/atlas.sum`.

## atlas.sum conflicts

If `tools/migrate/migrations/atlas.sum` conflicts or gets out of sync:

```bash
atlas migrate --env local hash
```

Then stage `atlas.sum`. Do not hand-edit checksum entries unless there is no other option.

## Ent and generated code

- `openmeter/ent/schema/*.go` is the source of truth.
- Never hand-edit `openmeter/ent/db/`.
- After schema-related rebased changes, run `make generate` before regenerating migrations.
- If generated files conflict, prefer regeneration over manual merge edits when possible.

## Shell/runtime notes

- If the ambient shell is missing Go/toolchain binaries, use:

```bash
nix develop --impure .#ci -c <command>
```

- Prefer direct command execution. Do not wrap commands in `sh -lc`, `bash -lc`, or similar helper shells when a direct invocation works. For environment variables, prefer `env KEY=value <command>` or `KEY=value <command>`.

- Atlas migration diffing uses the local dev environment and may require the direnv shell:

```bash
direnv exec . atlas migrate --env local diff <migration-name>
```

## Verification

Before continuing or finishing the rebase, run the smallest relevant test slice for the touched area.

For ledger work, the usual check is:

```bash
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./openmeter/ledger/...
```

If there is a known pre-existing failure, call it out explicitly and separate it from any new regression introduced by the rebase.

## Important reminders

- Never manually rewrite generated migration SQL unless the user explicitly wants that.
- Never keep two migrations that represent the same schema change at different timestamps.
- Prefer regenerating over “merging” generated artifacts.
- Stage only the resolved files for the current rebase step before `git rebase --continue`.
