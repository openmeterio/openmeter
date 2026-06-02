#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path


DEFAULT_ATLAS_SUM_PATH = "tools/migrate/migrations/atlas.sum"
MIGRATION_RECORD_PATTERN = re.compile(r"^([0-9]{14})_.+\.up\.sql h1:\S+$")


def fail(message: str) -> None:
    print(
        "tools/migrate/migrations/atlas.sum must only append new migration records.",
        file=sys.stderr,
    )
    print(message, file=sys.stderr)
    raise SystemExit(1)


def read_base_atlas_sum(base_revision: str, path: str) -> list[str]:
    try:
        result = subprocess.run(
            ["git", "show", f"{base_revision}:{path}"],
            check=True,
            capture_output=True,
            text=True,
        )
    except subprocess.CalledProcessError as error:
        stderr = error.stderr.strip()
        if stderr:
            print(stderr, file=sys.stderr)

        fail(f"Could not read {path} from base revision {base_revision}.")

    return result.stdout.splitlines()


def read_current_atlas_sum(path: str) -> list[str]:
    atlas_sum_path = Path(path)
    if not atlas_sum_path.is_file():
        fail(f"Current atlas.sum file does not exist at {path}.")

    return atlas_sum_path.read_text().splitlines()


def last_migration_timestamp(records: list[str]) -> str | None:
    timestamp = None

    for record in records:
        match = MIGRATION_RECORD_PATTERN.match(record)
        if match is not None:
            timestamp = match.group(1)

    return timestamp


def validate_appended_records(appended_records: list[str], previous_timestamp: str | None) -> None:
    for record in appended_records:
        match = MIGRATION_RECORD_PATTERN.match(record)
        if match is None:
            fail(f"Malformed appended atlas.sum migration record: {record}")

        timestamp = match.group(1)
        if previous_timestamp is not None and timestamp <= previous_timestamp:
            fail(
                f"Appended migration {record.split()[0]} must have a timestamp "
                f"newer than {previous_timestamp}."
            )

        previous_timestamp = timestamp


def check_atlas_sum_append_only(base_revision: str, path: str) -> None:
    base_lines = read_base_atlas_sum(base_revision, path)
    current_lines = read_current_atlas_sum(path)

    base_header = base_lines[:1]
    current_header = current_lines[:1]
    base_records = base_lines[1:]
    current_records = current_lines[1:]

    if current_records[: len(base_records)] != base_records:
        fail(
            "Existing migration records were modified, removed, reordered, "
            "or a new record was inserted before the previous end."
        )

    appended_records = current_records[len(base_records) :]
    if not appended_records:
        if current_header != base_header:
            fail("Atlas checksum header changed without any appended migration records.")

        return

    validate_appended_records(
        appended_records,
        previous_timestamp=last_migration_timestamp(base_records),
    )


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Check that atlas.sum only appends new migration records.",
    )
    parser.add_argument("base_revision")
    parser.add_argument("atlas_sum_path", nargs="?", default=DEFAULT_ATLAS_SUM_PATH)
    args = parser.parse_args()

    check_atlas_sum_append_only(args.base_revision, args.atlas_sum_path)


if __name__ == "__main__":
    main()
