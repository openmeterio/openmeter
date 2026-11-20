from __future__ import annotations

import importlib.util
import io
import unittest
from contextlib import redirect_stderr
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("check_pr_template.py")
MODULE_SPEC = importlib.util.spec_from_file_location(
    "check_pr_template",
    MODULE_PATH,
)
assert MODULE_SPEC is not None
assert MODULE_SPEC.loader is not None
check_pr_template = importlib.util.module_from_spec(MODULE_SPEC)
MODULE_SPEC.loader.exec_module(check_pr_template)


TEMPLATE = Path(__file__).parents[2] / "PULL_REQUEST_TEMPLATE.md"


def read_template_with_comments() -> str:
    return TEMPLATE.read_text()


class CheckPRBodyTemplateTest(unittest.TestCase):
    def assert_check_fails(self, body: str, expected_message: str) -> None:
        with redirect_stderr(io.StringIO()) as stderr:
            with self.assertRaises(SystemExit) as error:
                check_pr_template.check_pr_template(body)

        self.assertEqual(error.exception.code, 1)
        self.assertIn(expected_message, stderr.getvalue())

    def filled_section(self, heading: str, text: str) -> str:
        return f"## {heading}\n\n{text}\n"

    def test_allows_filled_template(self) -> None:
        body = "\n".join(
            [
                self.filled_section("What", "We changed the thing."),
                self.filled_section("Why", "It was broken."),
                self.filled_section("How", "By rewriting the parser."),
                self.filled_section("Notes for reviewer", "Mostly tests."),
            ]
        )

        check_pr_template.check_pr_template(body)

    def test_rejects_empty_body(self) -> None:
        self.assert_check_fails("", "description is empty")
        self.assert_check_fails("\n\n", "description is empty")

    def test_rejects_missing_section(self) -> None:
        body = "\n".join(
            [
                self.filled_section("What", "We changed the thing."),
                self.filled_section("How", "By rewriting the parser."),
            ]
        )

        self.assert_check_fails(body, "Missing required section: Why")

    def test_rejects_empty_section(self) -> None:
        body = "\n".join(
            [
                self.filled_section("What", "We changed the thing."),
                "## Why\n",
                self.filled_section("How", "By rewriting the parser."),
            ]
        )

        self.assert_check_fails(body, "Empty required section: Why")

    def test_rejects_whitespace_only_section(self) -> None:
        body = "\n".join(
            [
                self.filled_section("What", "We changed the thing."),
                "## Why\n\n   \n\t\n",
                self.filled_section("How", "By rewriting the parser."),
            ]
        )

        self.assert_check_fails(body, "Empty required section: Why")

    def test_rejects_fixes_placeholder_as_only_content(self) -> None:
        body = "\n".join(
            [
                self.filled_section("What", "We changed the thing."),
                self.filled_section("Why", "Fixes #(issue)"),
                self.filled_section("How", "By rewriting the parser."),
            ]
        )

        self.assert_check_fails(body, "only contains the placeholder")

    def test_allows_fixes_placeholder_with_additional_content(self) -> None:
        body = "\n".join(
            [
                self.filled_section("What", "We changed the thing."),
                self.filled_section("Why", "It was broken.\n\nFixes #(issue)"),
                self.filled_section("How", "By rewriting the parser."),
            ]
        )

        check_pr_template.check_pr_template(body)

    def test_ignores_html_comment_only_section(self) -> None:
        body = "\n".join(
            [
                self.filled_section("What", "We changed the thing."),
                "## Why\n\n<!-- motivation -->\n",
                self.filled_section("How", "By rewriting the parser."),
            ]
        )

        self.assert_check_fails(body, "Empty required section: Why")

    def test_section_content_after_comments_counts(self) -> None:
        body = "\n".join(
            [
                self.filled_section("What", "We changed the thing."),
                "## Why\n<!-- motivation -->\nIt was broken.\n",
                self.filled_section("How", "By rewriting the parser."),
            ]
        )

        check_pr_template.check_pr_template(body)

    def test_section_ends_at_next_heading(self) -> None:
        body = "\n".join(
            [
                self.filled_section("What", "We changed the thing."),
                self.filled_section("Why", "It was broken."),
                "## How\n",
                "## Notes for reviewer\n\nMostly tests.\n",
            ]
        )

        self.assert_check_fails(body, "Empty required section: How")

    def test_nested_heading_does_not_end_section(self) -> None:
        body = "\n".join(
            [
                self.filled_section("What", "We changed the thing."),
                self.filled_section("Why", "It was broken."),
                "## How\n\n- step one\n  - detail\n\nDone.\n",
            ]
        )

        check_pr_template.check_pr_template(body)

    def test_heading_with_trailing_spaces_matches(self) -> None:
        body = "\n".join(
            [
                self.filled_section("What", "We changed the thing."),
                "## Why   \n\nIt was broken.\n",
                self.filled_section("How", "By rewriting the parser."),
            ]
        )

        check_pr_template.check_pr_template(body)

    def test_lowercase_heading_is_rejected(self) -> None:
        body = "\n".join(
            [
                self.filled_section("What", "We changed the thing."),
                "## why\n\nIt was broken.\n",
                self.filled_section("How", "By rewriting the parser."),
            ]
        )

        self.assert_check_fails(body, "Missing required section: Why")

    def test_unmodified_template_fails(self) -> None:
        self.assert_check_fails(read_template_with_comments(), "Empty required section: What")


if __name__ == "__main__":
    unittest.main()
