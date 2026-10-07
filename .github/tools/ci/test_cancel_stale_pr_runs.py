import copy
import io
import json
import unittest
from contextlib import redirect_stdout
from unittest import mock
from urllib.error import HTTPError, URLError

import cancel_stale_pr_runs as controller
from cancel_stale_pr_runs import GitHubAPI, cancel_stale_pr_runs


SOURCE = {
    "id": 200,
    "workflow_id": 10,
    "run_number": 2,
    "run_attempt": 1,
    "event": "pull_request",
    "head_branch": "feature",
    "head_repository": {"id": 123, "owner": {"login": "openmeterio"}},
    "head_sha": "same-commit",
    "pull_requests": [{"number": 42}],
    "status": "queued",
}
PULL = {
    "number": 42,
    "created_at": "2026-10-01T00:00:00Z",
    "head": {"repo": {"id": 123}, "ref": "feature"},
    "base": {"repo": {"id": 123}},
    "user": {"login": "maintainer"},
}


class CancelStalePRRunsTest(unittest.TestCase):
    def setUp(self):
        self.source = copy.deepcopy(SOURCE)
        self.older = copy.deepcopy(SOURCE)
        # The actual incident had reversed run IDs and identical timestamps/SHA.
        # Only run_number determines dispatch order.
        self.older.update(id=300, run_number=1, status="in_progress")
        self.api = mock.Mock()
        self.api.pages.return_value = [self.older, self.source]

    def cancel(self):
        with redirect_stdout(io.StringIO()):
            cancel_stale_pr_runs(self.api, self.source, PULL)

    def test_newer_internal_run_cancels_older_run_of_same_commit(self):
        self.cancel()
        self.api.request.assert_called_once_with("actions/runs/300/cancel", method="POST")
        self.api.pages.assert_called_once_with(
            "actions/workflows/10/runs",
            {"event": "pull_request", "branch": "feature", "created": ">=2026-10-01T00:00:00Z"},
            "workflow_runs",
        )

    def test_delayed_older_controller_cancels_itself_not_newer_run(self):
        self.source = self.older
        self.api.pages.return_value = [self.older, SOURCE]
        self.cancel()
        self.api.request.assert_called_once_with("actions/runs/300/cancel", method="POST")

    def test_newer_completed_run_still_supersedes_older_active_run(self):
        newer = copy.deepcopy(SOURCE)
        newer.update(id=100, run_number=3, status="completed")
        self.api.pages.return_value = [self.older, self.source, newer]
        self.cancel()
        self.assertEqual(self.api.request.call_args_list, [
            mock.call("actions/runs/300/cancel", method="POST"),
            mock.call("actions/runs/200/cancel", method="POST"),
        ])

    def test_manual_rerun_of_older_run_does_not_supersede_newer_run(self):
        self.source = self.older
        self.source["run_attempt"] = 2
        self.api.pages.return_value = [self.source, SOURCE]
        self.cancel()
        self.api.request.assert_called_once_with("actions/runs/300/cancel", method="POST")

    def test_run_not_yet_visible_in_list_is_kept(self):
        self.api.pages.return_value = [self.older]
        self.cancel()
        self.api.request.assert_called_once_with("actions/runs/300/cancel", method="POST")

    def test_unrelated_runs_are_never_cancelled_or_used_as_newest(self):
        unrelated = []
        for changes in [
            {"workflow_id": 11},
            {"event": "push"},
            {"head_branch": "another-feature"},
            {"head_repository": {"id": 456}},
            {"pull_requests": [{"number": 43}]},
            {"pull_requests": []},
        ]:
            run = copy.deepcopy(SOURCE)
            run.update(id=999, run_number=99, **changes)
            unrelated.append(run)
        self.api.pages.return_value = [self.older, self.source, *unrelated]
        self.cancel()
        self.api.request.assert_called_once_with("actions/runs/300/cancel", method="POST")

    def test_fork_runs_are_left_alone(self):
        pull = copy.deepcopy(PULL)
        pull["head"]["repo"]["id"] = 456
        with redirect_stdout(io.StringIO()):
            cancel_stale_pr_runs(self.api, self.source, pull)
        self.api.pages.assert_not_called()
        self.api.request.assert_not_called()

    def test_source_from_another_repository_or_branch_is_left_alone(self):
        for changes in [{"head_repository": {"id": 456}}, {"head_branch": "other-feature"}]:
            with self.subTest(changes=changes):
                self.source = copy.deepcopy(SOURCE)
                self.source.update(changes)
                self.cancel()
                self.api.pages.assert_not_called()
                self.api.request.assert_not_called()

    def test_completed_older_runs_are_left_alone(self):
        self.older["status"] = "completed"
        self.cancel()
        self.api.request.assert_not_called()

    def test_non_pr_events_are_left_alone(self):
        self.source["event"] = "push"
        self.cancel()
        self.api.pages.assert_not_called()
        self.api.request.assert_not_called()

    def test_run_completing_during_cancellation_is_harmless(self):
        self.api.request.side_effect = HTTPError("url", 409, "Conflict", {}, io.BytesIO())
        self.addCleanup(self.api.request.side_effect.close)
        self.cancel()

    def test_permission_errors_are_reported(self):
        self.api.request.side_effect = HTTPError("url", 403, "Forbidden", {}, io.BytesIO())
        self.addCleanup(self.api.request.side_effect.close)
        with self.assertRaises(HTTPError):
            self.cancel()

    def test_runs_on_later_api_pages_are_included(self):
        api = GitHubAPI("owner/repo", "test-token", "https://api.github.com")
        with mock.patch.object(api, "request", side_effect=[
            {"workflow_runs": list(range(100))},
            {"workflow_runs": [100]},
        ]) as request:
            runs = list(api.pages("actions/workflows/10/runs", {"branch": "feature"}, "workflow_runs"))
        self.assertEqual(runs, list(range(101)))
        self.assertIn("page=2", request.call_args.args[0])


class CancellationEntryPointTest(unittest.TestCase):
    def test_current_calling_run_is_loaded_and_older_run_is_cancelled(self):
        api = mock.Mock()
        api.request.return_value = copy.deepcopy(SOURCE)
        older = copy.deepcopy(SOURCE)
        older.update(id=300, run_number=1, status="in_progress")
        api.pages.return_value = [older, SOURCE]
        with mock.patch.dict(controller.os.environ, {
            "GITHUB_EVENT_PATH": "event.json",
            "GITHUB_REPOSITORY": "openmeterio/openmeter",
            "GITHUB_TOKEN": "test-token",
            "GITHUB_API_URL": "https://api.github.com",
            "GITHUB_RUN_ID": "200",
        }, clear=True):
            with mock.patch("builtins.open", return_value=io.StringIO(json.dumps({"pull_request": PULL}))):
                with mock.patch.object(controller, "GitHubAPI", return_value=api) as constructor:
                    with redirect_stdout(io.StringIO()):
                        controller.main()
        constructor.assert_called_once_with("openmeterio/openmeter", "test-token", "https://api.github.com")
        self.assertEqual(api.request.call_args_list, [
            mock.call("actions/runs/200"),
            mock.call("actions/runs/300/cancel", method="POST"),
        ])

    def test_fork_dependabot_and_non_pr_events_skip_before_api_access(self):
        fork = copy.deepcopy(PULL)
        fork["head"]["repo"]["id"] = 456
        dependabot = copy.deepcopy(PULL)
        dependabot["user"]["login"] = "dependabot[bot]"
        deleted_fork = copy.deepcopy(PULL)
        deleted_fork["head"]["repo"] = None
        for event in [{}, {"pull_request": fork}, {"pull_request": dependabot}, {"pull_request": deleted_fork}]:
            with self.subTest(event=event):
                with mock.patch.dict(controller.os.environ, {"GITHUB_EVENT_PATH": "event.json"}, clear=True):
                    with mock.patch("builtins.open", return_value=io.StringIO(json.dumps(event))):
                        with mock.patch.object(controller, "GitHubAPI") as constructor:
                            with redirect_stdout(io.StringIO()):
                                controller.main()
                constructor.assert_not_called()


class GitHubAPIRequestTest(unittest.TestCase):
    def setUp(self):
        self.api = GitHubAPI("owner/repo", "test-token", "https://api.github.com")
        self.response = mock.MagicMock()
        self.response.__enter__.return_value.read.return_value = b'{"ok": true}'

    def test_recovers_from_transient_http_and_network_errors(self):
        errors = [
            HTTPError("url", code, "Temporary failure", {}, io.BytesIO())
            for code in (408, 500, 502, 503, 504)
        ] + [URLError("Connection refused"), TimeoutError(), ConnectionResetError()]
        for error in errors:
            if isinstance(error, HTTPError):
                self.addCleanup(error.close)
            with self.subTest(error=repr(error)):
                with mock.patch.object(controller, "urlopen", side_effect=[error, self.response]) as request:
                    with mock.patch("time.sleep") as sleep:
                        self.assertEqual(self.api.request("pulls"), {"ok": True})
                self.assertEqual(request.call_count, 2)
                sleep.assert_called_once_with(1)

    def test_stops_after_three_failed_attempts_and_preserves_error(self):
        for error in [HTTPError("url", 503, "Unavailable", {}, io.BytesIO()), URLError("Connection refused"), TimeoutError()]:
            if isinstance(error, HTTPError):
                self.addCleanup(error.close)
            with self.subTest(error=repr(error)):
                with mock.patch.object(controller, "urlopen", side_effect=error) as request:
                    with mock.patch("time.sleep") as sleep:
                        with self.assertRaises(type(error)) as raised:
                            self.api.request("pulls")
                self.assertIs(raised.exception, error)
                self.assertEqual(request.call_count, 3)
                self.assertEqual(sleep.call_args_list, [mock.call(1), mock.call(2)])

    def test_permission_and_other_client_errors_are_not_retried(self):
        for code in (401, 403, 404, 409, 422):
            with self.subTest(code=code):
                error = HTTPError("url", code, "Client error", {}, io.BytesIO())
                self.addCleanup(error.close)
                with mock.patch.object(controller, "urlopen", side_effect=error) as request:
                    with mock.patch("time.sleep") as sleep:
                        with self.assertRaises(HTTPError) as raised:
                            self.api.request("pulls")
                self.assertIs(raised.exception, error)
                request.assert_called_once()
                sleep.assert_not_called()

    def test_timeout_reading_response_is_retried(self):
        interrupted = mock.MagicMock()
        interrupted.__enter__.return_value.read.side_effect = TimeoutError()
        with mock.patch.object(controller, "urlopen", side_effect=[interrupted, self.response]) as request:
            with mock.patch("time.sleep"):
                self.assertEqual(self.api.request("pulls"), {"ok": True})
        self.assertEqual(request.call_count, 2)
        interrupted.__exit__.assert_called_once()

    def test_cancellation_conflict_after_lost_response_is_harmless(self):
        older = copy.deepcopy(SOURCE)
        older.update(id=300, run_number=1, status="in_progress")
        conflict = HTTPError("url", 409, "Already cancelled", {}, io.BytesIO())
        self.addCleanup(conflict.close)
        with mock.patch.object(self.api, "pages", return_value=[older, SOURCE]):
            with mock.patch.object(controller, "urlopen", side_effect=[
                TimeoutError(), conflict,
            ]) as request:
                with mock.patch("time.sleep"):
                    with redirect_stdout(io.StringIO()):
                        cancel_stale_pr_runs(self.api, SOURCE, PULL)
        self.assertEqual(request.call_count, 2)
        for call in request.call_args_list:
            self.assertEqual(call.args[0].get_method(), "POST")
            self.assertEqual(call.args[0].full_url, "https://api.github.com/repos/owner/repo/actions/runs/300/cancel")


if __name__ == "__main__":
    unittest.main()
