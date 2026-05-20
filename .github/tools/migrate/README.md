# Migration CI Checks

This directory contains GitHub Actions helper code for migration-specific PR
checks.

`check_atlas_sum_append_only.py` protects `tools/migrate/migrations/atlas.sum`.
Atlas updates the first `h1:` line when the migration directory hash changes, but
existing migration record lines should remain immutable once they are on the base
branch. The check compares the PR copy of `atlas.sum` against the pull request
base revision and enforces that:

- existing migration records are unchanged and stay in the same order
- new migration records are appended after the previous end of the file
- appended migration timestamps are strictly newer than the previous last
  migration timestamp
- the Atlas header checksum changes only when records are appended

Run the unit tests locally with:

```bash
python3 -m unittest discover -s .github/tools/migrate -p 'test_*.py'
```

Run the same append-only check that CI runs with:

```bash
python3 .github/tools/migrate/check_atlas_sum_append_only.py main
```

Pass a different base revision when testing a branch against something other
than `main`.
