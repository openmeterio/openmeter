#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

REQUIRED_SECTIONS = ("What", "Why", "How")

FIXES_PLACEHOLDER = "Fixes #(issue)"


def fail(message: str) -> None:
    print("The pull request description does not follow the PR template.", file=sys.stderr)
    print(message, file=sys.stderr)
    raise SystemExit(1)


def strip_html_comments(text: str) -> str:
    return re.sub(r"<!--.*?-->", "", text, flags=re.DOTALL)


def extract_section(body: str, heading: str) -> str | None:
    pattern = re.compile(
        rf"^##\s+{re.escape(heading)}\s*$",
        flags=re.MULTILINE,
    )
    match = pattern.search(body)
    if match is None:
        return None

    rest = body[match.end():]
    next_heading = re.search(r"^##\s+", rest, flags=re.MULTILINE)
    if next_heading is not None:
        rest = rest[: next_heading.start()]

    return strip_html_comments(rest).strip()


def check_section(body: str, heading: str) -> None:
    section = extract_section(body, heading)
    if section is None:
        fail(f"Missing required section: {heading} (## {heading})")

    if not section:
        fail(f"Empty required section: {heading}")

    if section == FIXES_PLACEHOLDER:
        fail(f"Required section {heading} only contains the placeholder {FIXES_PLACEHOLDER}")


def check_pr_template(body: str) -> None:
    # The template ships inside HTML comments; authors who leave them in place
    # keep the placeholder text, so comments must go before the emptiness check.
    if not strip_html_comments(body).strip():
        fail("The pull request description is empty")

    for heading in REQUIRED_SECTIONS:
        check_section(body, heading)


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Check that the pull request description follows the What/Why/How template.",
    )
    parser.add_argument("body_path", help="Path to the pull request body, or - for stdin.")
    args = parser.parse_args()

    if args.body_path == "-":
        body = sys.stdin.read()
    else:
        body_path = Path(args.body_path)
        if not body_path.is_file():
            fail(f"PR body file does not exist at {args.body_path}.")
        body = body_path.read_text()

    check_pr_template(body)


if __name__ == "__main__":
    main()
