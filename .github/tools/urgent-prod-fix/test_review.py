import copy
import io
import json
import unittest
from unittest.mock import Mock, patch
from urllib.error import HTTPError, URLError

import review


class EmergencyReviewTest(unittest.TestCase):
    def setUp(self):
        self.bot = "emergency[bot]"
        self.path = "/repos/openmeterio/openmeter/pulls/123"
        self.pr = {
            "number": 123,
            "state": "open",
            "draft": False,
            "merged": False,
            "merge_commit_sha": "b" * 40,
            "base": {"ref": "main", "repo": {"full_name": review.REPOSITORY}},
            "head": {"sha": "a" * 40, "repo": {"full_name": review.REPOSITORY}},
            "user": {"login": "author", "type": "User"},
            "labels": [{"name": review.LABEL}],
        }
        self.event = {
            "number": 123,
            "action": "labeled",
            "repository": {"full_name": review.REPOSITORY},
            "pull_request": copy.deepcopy(self.pr),
            "label": {"name": review.LABEL},
            "sender": {"login": "turip", "type": "User"},
        }
        self.approval = {
            "id": 456,
            "user": {"login": self.bot, "type": "Bot"},
            "state": "APPROVED",
            "commit_id": self.pr["head"]["sha"],
            "body": review.MARKER,
            "html_url": "https://github.com/openmeterio/openmeter/pull/123#pullrequestreview-456",
        }
        self.github = Mock()
        self.github.reviews.return_value = []
        self.github.request.side_effect = self.respond

    def respond(self, method, path, body=None):
        if method == "GET" and path == self.path:
            return copy.deepcopy(self.pr)
        if method == "GET" and "/memberships/" in path:
            return {"state": "active"}
        if method == "POST" and path == f"{self.path}/reviews":
            return copy.deepcopy(self.approval)
        if method == "PUT" and path.endswith("/dismissals"):
            return {}
        if method == "DELETE" and path == "/repos/openmeterio/openmeter/issues/123/labels/urgent-prod-fix":
            self.pr["labels"] = [label for label in self.pr["labels"] if label["name"] != review.LABEL]
            return copy.deepcopy(self.pr["labels"])
        raise AssertionError((method, path, body))

    def test_authorized_request_approves_exact_revision_without_merge(self):
        review.process(self.github, self.event, self.bot, "1")
        writes = [call for call in self.github.request.call_args_list if call.args[0] != "GET"]
        self.assertEqual(len(writes), 1)
        self.assertEqual(writes[0].args[:2], ("POST", f"{self.path}/reviews"))
        self.assertEqual(writes[0].args[2]["commit_id"], "a" * 40)
        self.assertEqual(writes[0].args[2]["event"], "APPROVE")
        self.github.request.assert_any_call(
            "GET", "/orgs/openmeterio/teams/openmeter-oncall/memberships/turip"
        )
        self.github.request.assert_any_call("GET", "/orgs/openmeterio/memberships/author")

    def test_rejects_ineligible_prs_without_writing(self):
        for change in [
            {"draft": True},
            {"state": "closed"},
            {"labels": []},
            {"head": {"sha": "b" * 40, "repo": {"full_name": review.REPOSITORY}}},
            {"head": {"sha": "a" * 40, "repo": {"full_name": "outsider/openmeter"}}},
            {"head": {"sha": "a" * 40, "repo": None}},
            {"base": {"ref": "release", "repo": {"full_name": review.REPOSITORY}}},
            {"user": {"login": "bot", "type": "Bot"}},
        ]:
            with self.subTest(change=change):
                self.pr = copy.deepcopy(self.event["pull_request"])
                self.pr.update(change)
                self.github.reset_mock()
                with self.assertRaises(ValueError):
                    review.process(self.github, self.event, self.bot, "1")
                self.assertTrue(all(c.args[0] == "GET" for c in self.github.request.call_args_list))

    def test_membership_failures_deny_approval(self):
        for rejected_path in ["teams/openmeter-oncall/memberships/turip", "memberships/author"]:
            for failure in ["pending", 403, 404, 500]:
                with self.subTest(path=rejected_path, failure=failure):
                    def respond(method, path, body=None):
                        if path.endswith(rejected_path):
                            if failure == "pending":
                                return {"state": "pending"}
                            raise HTTPError(path, failure, "denied", {}, None)
                        return self.respond(method, path, body)

                    self.github.reset_mock()
                    self.github.request.side_effect = respond
                    with self.assertRaises((ValueError, HTTPError)):
                        review.process(self.github, self.event, self.bot, "1")
                    self.assertTrue(all(c.args[0] == "GET" for c in self.github.request.call_args_list))

    def test_rerun_cannot_reuse_request(self):
        with self.assertRaises(ValueError):
            review.process(self.github, self.event, self.bot, "2")
        self.assertTrue(all(c.args[0] == "GET" for c in self.github.request.call_args_list))

    def test_bot_cannot_request_approval(self):
        self.event["sender"]["type"] = "Bot"
        with self.assertRaises(ValueError):
            review.process(self.github, self.event, self.bot, "1")

    def test_changed_revision_during_submission_is_dismissed(self):
        def respond(method, path, body=None):
            result = self.respond(method, path, body)
            if method == "POST":
                self.pr["head"]["sha"] = "b" * 40
            return result

        self.github.request.side_effect = respond
        with self.assertRaises(ValueError):
            review.process(self.github, self.event, self.bot, "1")
        self.assertEqual(self.github.request.call_args.args[:2], ("PUT", f"{self.path}/reviews/456/dismissals"))

    def test_push_or_label_removal_dismisses_only_emergency_approval(self):
        for action in ["synchronize", "unlabeled"]:
            with self.subTest(action=action):
                self.event["action"] = action
                self.pr["labels"] = []
                human_review = copy.deepcopy(self.approval)
                human_review["id"] = 789
                human_review["user"] = {"login": "turip", "type": "User"}
                self.github.reviews.return_value = [self.approval, human_review]
                self.github.reset_mock()
                review.process(self.github, self.event, self.bot, "1")
                writes = [c for c in self.github.request.call_args_list if c.args[0] != "GET"]
                self.assertEqual(len(writes), 1)
                self.assertEqual(writes[0].args[1], f"{self.path}/reviews/456/dismissals")

    def test_delayed_invalidation_does_not_dismiss_current_approval(self):
        self.event["action"] = "unlabeled"
        self.github.reviews.return_value = [self.approval]
        review.process(self.github, self.event, self.bot, "1")
        self.assertTrue(all(c.args[0] == "GET" for c in self.github.request.call_args_list))

    def test_push_removes_only_emergency_label_and_dismisses_approval(self):
        self.event["action"] = "synchronize"
        self.pr["labels"].append({"name": "kind/bug"})
        self.github.reviews.return_value = [self.approval]
        review.process(self.github, self.event, self.bot, "1")
        self.assertEqual(self.pr["labels"], [{"name": "kind/bug"}])
        writes = [c for c in self.github.request.call_args_list if c.args[0] != "GET"]
        self.assertEqual([c.args[:2] for c in writes], [
            ("DELETE", "/repos/openmeterio/openmeter/issues/123/labels/urgent-prod-fix"),
            ("PUT", f"{self.path}/reviews/456/dismissals"),
        ])

    def test_push_without_label_or_after_close_does_not_remove_labels(self):
        self.event["action"] = "synchronize"
        for change in [{"labels": []}, {"state": "closed"}]:
            with self.subTest(change=change):
                self.pr = copy.deepcopy(self.event["pull_request"])
                self.pr.update(change)
                self.github.reset_mock()
                review.process(self.github, self.event, self.bot, "1")
                self.assertTrue(all(c.args[0] == "GET" for c in self.github.request.call_args_list))

    def test_concurrent_label_removal_still_dismisses_approval(self):
        self.event["action"] = "synchronize"
        self.github.reviews.return_value = [self.approval]

        def respond(method, path, body=None):
            if method == "DELETE":
                raise HTTPError(path, 404, "already removed", {}, io.BytesIO())
            return self.respond(method, path, body)

        self.github.request.side_effect = respond
        review.process(self.github, self.event, self.bot, "1")
        self.assertEqual(self.github.request.call_args.args[:2], ("PUT", f"{self.path}/reviews/456/dismissals"))

    def test_existing_approval_is_not_duplicated(self):
        self.github.reviews.return_value = [self.approval]
        review.process(self.github, self.event, self.bot, "1")
        self.assertTrue(all(c.args[0] == "GET" for c in self.github.request.call_args_list))

    @patch.object(review, "notify_slack")
    def test_only_merged_emergency_revision_notifies(self, notify):
        self.event["action"] = "closed"
        self.github.reviews.return_value = [self.approval]
        review.process(self.github, self.event, self.bot, "1")
        notify.assert_not_called()
        self.event["pull_request"]["merged"] = True
        self.pr["merged"] = True
        self.pr["state"] = "closed"
        self.pr["labels"] = []
        review.process(self.github, self.event, self.bot, "1")
        notify.assert_called_once_with(self.pr, self.approval)
        notify.reset_mock()
        self.pr["head"]["sha"] = "c" * 40
        review.process(self.github, self.event, self.bot, "1")
        notify.assert_not_called()

    @patch.dict(review.os.environ, {"URGENT_FIX_SLACK_WEBHOOK_URL": "https://hooks.slack.com/services/secret"})
    @patch.object(review, "urlopen")
    def test_slack_excludes_untrusted_mentions_and_hides_webhook_errors(self, urlopen):
        self.pr["title"] = "<!channel>"
        urlopen.return_value.__enter__.return_value = io.BytesIO(b"ok")
        review.notify_slack(self.pr, self.approval)
        payload = json.loads(urlopen.call_args.args[0].data)
        self.assertNotIn("<!channel>", payload["text"])
        urlopen.side_effect = URLError("https://hooks.slack.com/services/secret")
        with self.assertRaises(ValueError) as error:
            review.notify_slack(self.pr, self.approval)
        self.assertNotIn("secret", str(error.exception))

    def test_review_listing_paginates(self):
        github = review.GitHub("unused")
        github.request = Mock(side_effect=[[{}] * 100, [self.approval]])
        self.assertEqual(len(list(github.reviews(self.path))), 101)


if __name__ == "__main__":
    unittest.main()
