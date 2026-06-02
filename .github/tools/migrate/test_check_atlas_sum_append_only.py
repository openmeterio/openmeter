from __future__ import annotations

import importlib.util
import io
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest import mock


MODULE_PATH = Path(__file__).with_name("check_atlas_sum_append_only.py")
MODULE_SPEC = importlib.util.spec_from_file_location(
    "check_atlas_sum_append_only",
    MODULE_PATH,
)
assert MODULE_SPEC is not None
assert MODULE_SPEC.loader is not None
check_atlas_sum_append_only = importlib.util.module_from_spec(MODULE_SPEC)
MODULE_SPEC.loader.exec_module(check_atlas_sum_append_only)


class CheckAtlasSumAppendOnlyTest(unittest.TestCase):
    def run_check(self, base_atlas_sum: str, current_atlas_sum: str) -> None:
        with tempfile.TemporaryDirectory() as tmp_dir:
            atlas_sum_path = Path(tmp_dir) / "atlas.sum"
            atlas_sum_path.write_text(current_atlas_sum)

            with mock.patch.object(
                check_atlas_sum_append_only.subprocess,
                "run",
                return_value=subprocess.CompletedProcess(
                    args=["git", "show", f"base:{atlas_sum_path}"],
                    returncode=0,
                    stdout=base_atlas_sum,
                    stderr="",
                ),
            ) as git_show:
                check_atlas_sum_append_only.check_atlas_sum_append_only(
                    "base",
                    str(atlas_sum_path),
                )

            git_show.assert_called_once_with(
                ["git", "show", f"base:{atlas_sum_path}"],
                check=True,
                capture_output=True,
                text=True,
            )

    def assert_check_fails(self, base_atlas_sum: str, current_atlas_sum: str) -> None:
        with mock.patch("sys.stderr", new_callable=io.StringIO):
            with self.assertRaises(SystemExit) as error:
                self.run_check(base_atlas_sum, current_atlas_sum)

        self.assertEqual(error.exception.code, 1)

    def test_allows_unchanged_atlas_sum(self) -> None:
        atlas_sum = "\n".join(
            [
                "h1:base",
                "20260519132345_reset-sync-state.up.sql h1:old=",
                "",
            ]
        )

        self.run_check(atlas_sum, atlas_sum)

    def test_allows_newer_records_appended_after_existing_records(self) -> None:
        base_atlas_sum = "\n".join(
            [
                "h1:base",
                "20260519132345_reset-sync-state.up.sql h1:old=",
                "",
            ]
        )
        current_atlas_sum = "\n".join(
            [
                "h1:changed",
                "20260519132345_reset-sync-state.up.sql h1:old=",
                "20260520120000_newer.up.sql h1:new=",
                "",
            ]
        )

        self.run_check(base_atlas_sum, current_atlas_sum)

    def test_rejects_changed_existing_record(self) -> None:
        base_atlas_sum = "\n".join(
            [
                "h1:base",
                "20260519132345_reset-sync-state.up.sql h1:old=",
                "",
            ]
        )
        current_atlas_sum = "\n".join(
            [
                "h1:changed",
                "20260519132345_reset-sync-state.up.sql h1:changed=",
                "20260520120000_newer.up.sql h1:new=",
                "",
            ]
        )

        self.assert_check_fails(base_atlas_sum, current_atlas_sum)

    def test_rejects_header_change_without_appended_record(self) -> None:
        base_atlas_sum = "\n".join(
            [
                "h1:base",
                "20260519132345_reset-sync-state.up.sql h1:old=",
                "",
            ]
        )
        current_atlas_sum = "\n".join(
            [
                "h1:changed",
                "20260519132345_reset-sync-state.up.sql h1:old=",
                "",
            ]
        )

        self.assert_check_fails(base_atlas_sum, current_atlas_sum)

    def test_rejects_appended_record_with_older_timestamp(self) -> None:
        base_atlas_sum = "\n".join(
            [
                "h1:base",
                "20260519132345_reset-sync-state.up.sql h1:old=",
                "",
            ]
        )
        current_atlas_sum = "\n".join(
            [
                "h1:changed",
                "20260519132345_reset-sync-state.up.sql h1:old=",
                "20260518120000_older.up.sql h1:new=",
                "",
            ]
        )

        self.assert_check_fails(base_atlas_sum, current_atlas_sum)

    def test_rejects_malformed_appended_record(self) -> None:
        base_atlas_sum = "\n".join(
            [
                "h1:base",
                "20260519132345_reset-sync-state.up.sql h1:old=",
                "",
            ]
        )
        current_atlas_sum = "\n".join(
            [
                "h1:changed",
                "20260519132345_reset-sync-state.up.sql h1:old=",
                "not-a-migration h1:new=",
                "",
            ]
        )

        self.assert_check_fails(base_atlas_sum, current_atlas_sum)


if __name__ == "__main__":
    unittest.main()
