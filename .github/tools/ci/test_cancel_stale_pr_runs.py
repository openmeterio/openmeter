import copy
import io
import unittest
from contextlib import redirect_stdout
from unittest import mock
from urllib.error import HTTPError

from cancel_stale_pr_runs import GitHubAPI, cancel_stale_pr_runs


SOURCE = {
    "id": 200,
    "workflow_id": 10,
    "run_number": 2,
    "run_attempt": 1,
    "event": "pull_request",
    "head_branch": "feature",
    "head_repository": {"id": 123, "owner": {"login": "contributor"}},
    "head_sha": "same-commit",
    "pull_requests": [],
    "status": "queued",
}
PULL = {
    "number": 42,
    "created_at": "2026-10-01T00:00:00Z",
    "head": {"repo": {"id": 123}, "ref": "feature"},
}


class CancelStalePRRunsTest(unittest.TestCase):
    def setUp(self):
        self.source = copy.deepcopy(SOURCE)
        self.older = copy.deepcopy(SOURCE)
        # The actual incident had reversed run IDs and identical timestamps/SHA.
        # Only run_number determines dispatch order.
        self.older.update(id=300, run_number=1, status="in_progress")
        self.api = mock.Mock()
        self.api.pages.side_effect = [[copy.deepcopy(PULL)], [self.older, self.source]]

    def cancel(self):
        with redirect_stdout(io.StringIO()):
            cancel_stale_pr_runs(self.api, self.source)

    def test_newer_fork_run_cancels_older_run_of_same_commit(self):
        self.cancel()
        self.api.request.assert_called_once_with("actions/runs/300/cancel", method="POST")
        self.assertEqual(self.api.pages.call_args_list[0].args, (
            "pulls", {"head": "contributor:feature", "state": "open"},
        ))

    def test_delayed_older_controller_cancels_itself_not_newer_run(self):
        self.source = self.older
        self.api.pages.side_effect = [[copy.deepcopy(PULL)], [self.older, SOURCE]]
        self.cancel()
        self.api.request.assert_called_once_with("actions/runs/300/cancel", method="POST")

    def test_newer_completed_run_still_supersedes_older_active_run(self):
        newer = copy.deepcopy(SOURCE)
        newer.update(id=100, run_number=3, status="completed")
        self.api.pages.side_effect = [[copy.deepcopy(PULL)], [self.older, self.source, newer]]
        self.cancel()
        self.assertEqual(self.api.request.call_args_list, [
            mock.call("actions/runs/300/cancel", method="POST"),
            mock.call("actions/runs/200/cancel", method="POST"),
        ])

    def test_manual_rerun_of_older_run_does_not_supersede_newer_run(self):
        self.source = self.older
        self.source["run_attempt"] = 2
        self.api.pages.side_effect = [[copy.deepcopy(PULL)], [self.source, SOURCE]]
        self.cancel()
        self.api.request.assert_called_once_with("actions/runs/300/cancel", method="POST")

    def test_run_not_yet_visible_in_list_is_kept(self):
        self.api.pages.side_effect = [[copy.deepcopy(PULL)], [self.older]]
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
        ]:
            run = copy.deepcopy(SOURCE)
            run.update(id=999, run_number=99, **changes)
            unrelated.append(run)
        self.api.pages.side_effect = [[copy.deepcopy(PULL)], [self.older, self.source, *unrelated]]
        self.cancel()
        self.api.request.assert_called_once_with("actions/runs/300/cancel", method="POST")

    def test_fork_branch_with_multiple_open_prs_is_left_alone(self):
        another_pull = copy.deepcopy(PULL)
        another_pull["number"] = 43
        self.api.pages.side_effect = [[copy.deepcopy(PULL), another_pull]]
        self.cancel()
        self.api.request.assert_not_called()

    def test_pr_metadata_disambiguates_same_repo_branch(self):
        self.source["pull_requests"] = [{"number": 42}]
        another_pull = copy.deepcopy(PULL)
        another_pull["number"] = 43
        self.api.pages.side_effect = [[copy.deepcopy(PULL), another_pull], [self.older, self.source]]
        self.cancel()
        self.api.request.assert_called_once_with("actions/runs/300/cancel", method="POST")

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
        self.api.request.side_effect = HTTPError("url", 409, "Conflict", {}, None)
        self.cancel()

    def test_permission_errors_are_reported(self):
        self.api.request.side_effect = HTTPError("url", 403, "Forbidden", {}, None)
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


if __name__ == "__main__":
    unittest.main()
