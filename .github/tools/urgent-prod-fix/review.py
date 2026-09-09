from __future__ import annotations

import json
import os
import sys
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import quote
from urllib.request import Request, urlopen


REPOSITORY = "openmeterio/openmeter"
LABEL = "urgent-prod-fix"
MARKER = "<!-- openmeter-urgent-prod-fix -->"


class GitHub:
    def __init__(self, token: str):
        self.token = token

    def request(self, method: str, path: str, body=None):
        request = Request(
            f"https://api.github.com{path}",
            data=None if body is None else json.dumps(body).encode(),
            method=method,
            headers={
                "Authorization": f"Bearer {self.token}",
                "Accept": "application/vnd.github+json",
                "Content-Type": "application/json",
                "X-GitHub-Api-Version": "2022-11-28",
            },
        )
        with urlopen(request, timeout=30) as response:
            return json.load(response)

    def reviews(self, path: str):
        page = 1
        while True:
            reviews = self.request("GET", f"{path}/reviews?per_page=100&page={page}")
            yield from reviews
            if len(reviews) < 100:
                return
            page += 1


def require_member(github, path: str):
    if github.request("GET", path).get("state") != "active":
        raise ValueError("An active organization/team membership is required")


def eligible(pr, sha: str) -> bool:
    return (
        pr["state"] == "open"
        and not pr["draft"]
        and pr["base"]["repo"]["full_name"] == REPOSITORY
        and pr["base"]["ref"] == "main"
        and (pr["head"]["repo"] or {}).get("full_name") == REPOSITORY
        and pr["head"]["sha"] == sha
        and pr["user"]["type"] == "User"
        and any(label["name"] == LABEL for label in pr["labels"])
    )


def emergency_reviews(github, path: str, bot: str):
    return [
        review
        for review in github.reviews(path)
        if review["user"]["login"] == bot
        and review["user"]["type"] == "Bot"
        and review["state"] == "APPROVED"
        and (review.get("body") or "").startswith(MARKER)
    ]


def dismiss(github, path: str, review):
    github.request(
        "PUT",
        f"{path}/reviews/{review['id']}/dismissals",
        {"message": "Emergency request invalidated; reapply urgent-prod-fix to request a new approval."},
    )


def notify_slack(pr, review):
    webhook = os.environ["URGENT_FIX_SLACK_WEBHOOK_URL"]
    if not webhook:
        raise ValueError("URGENT_FIX_SLACK_WEBHOOK_URL is not configured")
    # PR titles and bodies are untrusted Slack markup, including possible mentions.
    text = (
        f"Urgent production fix merged: https://github.com/{REPOSITORY}/pull/{pr['number']}\n"
        f"Merged commit: {pr['merge_commit_sha']}\n"
        f"Emergency authorization: {review['html_url']}\n"
        "Engineering: please review the changes post-merge."
    )
    request = Request(
        webhook,
        data=json.dumps({"text": text, "unfurl_links": False, "unfurl_media": False}).encode(),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    # Network exceptions can contain the secret webhook URL.
    try:
        with urlopen(request, timeout=30) as response:
            if response.read().decode().strip() != "ok":
                raise ValueError("Slack rejected the notification; rerun this closed-event job")
    except (HTTPError, URLError, TimeoutError):
        raise ValueError("Slack notification failed; rerun this closed-event job") from None


def process(github, event, bot: str, run_attempt: str):
    if event["repository"]["full_name"] != REPOSITORY:
        raise ValueError("Unexpected repository")
    action = event["action"]
    if action not in {"labeled", "unlabeled", "synchronize", "converted_to_draft", "closed"}:
        return
    if action in {"labeled", "unlabeled"} and event["label"]["name"] != LABEL:
        return

    path = f"/repos/{REPOSITORY}/pulls/{int(event['number'])}"
    pr = github.request("GET", path)
    reviews = emergency_reviews(github, path, bot)

    if action == "closed":
        if event["pull_request"]["merged"] and pr["merged"]:
            matching = [r for r in reviews if r["commit_id"] == pr["head"]["sha"]]
            if matching:
                notify_slack(pr, matching[-1])
        return

    if action != "labeled":
        if action == "synchronize" and pr["state"] == "open":
            if any(label["name"] == LABEL for label in pr["labels"]):
                try:
                    github.request(
                        "DELETE",
                        f"/repos/{REPOSITORY}/issues/{int(event['number'])}/labels/{LABEL}",
                    )
                except HTTPError as error:
                    if error.code != 404:
                        raise
                pr["labels"] = [label for label in pr["labels"] if label["name"] != LABEL]

        for review in reviews:
            if not eligible(pr, review["commit_id"]):
                dismiss(github, path, review)
        return

    if run_attempt != "1":
        raise ValueError("Reapply urgent-prod-fix instead of rerunning an approval request")
    sha = event["pull_request"]["head"]["sha"]
    if not eligible(pr, sha):
        raise ValueError("Request requires a labeled, internal, non-draft PR to main at the original revision")
    if event["sender"]["type"] != "User":
        raise ValueError("Only a human openmeter-oncall team member can request approval")
    requester = quote(event["sender"]["login"], safe="")
    author = quote(pr["user"]["login"], safe="")
    require_member(github, f"/orgs/openmeterio/teams/openmeter-oncall/memberships/{requester}")
    require_member(github, f"/orgs/openmeterio/memberships/{author}")
    pr = github.request("GET", path)
    if not eligible(pr, sha):
        raise ValueError("PR changed during authorization; reapply urgent-prod-fix")
    if any(review["commit_id"] == sha for review in reviews):
        return
    review = github.request(
        "POST",
        f"{path}/reviews",
        {
            "event": "APPROVE",
            "commit_id": sha,
            "body": (
                f"{MARKER}\nEmergency approval requested by @{requester} "
                f"(verified member of @openmeterio/openmeter-oncall) for `{sha}`.\n\n"
                "Human code review is deferred until after merge. Required CI checks still apply."
            ),
        },
    )
    current = github.request("GET", path)
    if not current["merged"] and not eligible(current, sha):
        dismiss(github, path, review)
        raise ValueError("PR changed while approval was submitted; emergency approval dismissed")


if __name__ == "__main__":
    try:
        process(
            GitHub(os.environ["URGENT_FIX_TOKEN"]),
            json.loads(Path(os.environ["GITHUB_EVENT_PATH"]).read_text()),
            os.environ["URGENT_FIX_BOT"],
            os.environ["GITHUB_RUN_ATTEMPT"],
        )
    except (ValueError, HTTPError, URLError, TimeoutError) as error:
        print(f"Emergency review failed: {error}", file=sys.stderr)
        sys.exit(1)
