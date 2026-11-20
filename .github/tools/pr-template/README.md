# PR Template Check

This directory contains the GitHub Actions helper code for the pull request
description check.

`check_pr_template.py` validates that the PR description follows the What / Why
/ How template. It reads the pull request body and enforces that every required
section:

- exists under its expected heading
- is not empty
- does not only contain the template's placeholder comment or the unresolved
  `Fixes #(issue)` marker

HTML comments are ignored, so authors who delete the placeholder comments are
not penalized. The `Notes for reviewer` section is optional.

Run the unit tests locally with:

```bash
python3 -m unittest discover -s .github/tools/pr-template -p 'test_*.py'
```

Run the same check that CI runs with:

```bash
python3 .github/tools/pr-template/check_pr_template.py pr-body.md
```

The script takes the PR body as a file (or `-` for stdin) so the workflow can
pass the event payload without exposing it to shell interpolation.
