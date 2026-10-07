"""Cancel older PR runs by workflow run number, independent of queue arrival order."""

import json
import os
from urllib.error import HTTPError
from urllib.parse import urlencode
from urllib.request import Request, urlopen


class GitHubAPI:
    def __init__(self, repository, token, api_url):
        self.base_url = f"{api_url}/repos/{repository}"
        self.token = token

    def request(self, path, method="GET"):
        request = Request(
            f"{self.base_url}/{path}",
            method=method,
            headers={
                "Authorization": f"Bearer {self.token}",
                "Accept": "application/vnd.github+json",
                "X-GitHub-Api-Version": "2022-11-28",
            },
        )
        with urlopen(request, timeout=30) as response:
            body = response.read()
            return json.loads(body) if body else None

    def pages(self, path, parameters, key=None):
        page = 1
        while True:
            query = urlencode({**parameters, "per_page": 100, "page": page})
            response = self.request(f"{path}?{query}")
            items = response[key] if key else response
            yield from items
            if len(items) < 100:
                return
            page += 1


def cancel_stale_pr_runs(api, source):
    if source["event"] != "pull_request" or not source.get("head_repository"):
        return

    # Fork runs omit pull_requests in the Actions API. Resolve their source
    # branch through the PR API rather than assuming branch names are unique.
    head_repository = source["head_repository"]
    head = f"{head_repository['owner']['login']}:{source['head_branch']}"
    pulls = [
        pull for pull in api.pages("pulls", {"head": head, "state": "open"})
        if pull["head"]["repo"]
        and pull["head"]["repo"]["id"] == head_repository["id"]
        and pull["head"]["ref"] == source["head_branch"]
    ]
    source_numbers = {pull["number"] for pull in source["pull_requests"]}
    if source_numbers:
        pulls = [pull for pull in pulls if pull["number"] in source_numbers]
    if len(pulls) != 1:
        print("Skipping cancellation: run cannot be assigned to one open PR")
        return

    pull = pulls[0]
    runs = [
        run for run in api.pages(
            f"actions/workflows/{source['workflow_id']}/runs",
            {
                "event": "pull_request",
                "branch": source["head_branch"],
                "created": f">={pull['created_at']}",
            },
            "workflow_runs",
        )
        if run["workflow_id"] == source["workflow_id"]
        and run["event"] == "pull_request"
        and run["head_branch"] == source["head_branch"]
        and run.get("head_repository")
        and run["head_repository"]["id"] == head_repository["id"]
        and (not run["pull_requests"] or any(
            candidate["number"] == pull["number"]
            for candidate in run["pull_requests"]
        ))
    ]
    # The event's run may not yet appear in the list endpoint. Use its latest
    # state so a delayed requested event cannot cancel a completed attempt.
    runs = [run for run in runs if run["id"] != source["id"]] + [source]
    newest = max(runs, key=lambda run: run["run_number"])
    print(f"Keeping workflow run {newest['run_number']} for PR {pull['number']}")
    for run in runs:
        if run["run_number"] >= newest["run_number"] or run["status"] == "completed":
            continue
        try:
            api.request(f"actions/runs/{run['id']}/cancel", method="POST")
            print(f"Cancelled older workflow run {run['run_number']}")
        except HTTPError as error:
            if error.code != 409:
                raise
            # A run may complete or receive another controller's cancellation
            # after the list request. Both outcomes need no further action.
            print(f"Run {run['run_number']} is no longer cancellable")


if __name__ == "__main__":
    with open(os.environ["GITHUB_EVENT_PATH"]) as event_file:
        event = json.load(event_file)
    api = GitHubAPI(
        os.environ["GITHUB_REPOSITORY"],
        os.environ["GITHUB_TOKEN"],
        os.environ["GITHUB_API_URL"],
    )
    source = api.request(f"actions/runs/{event['workflow_run']['id']}")
    cancel_stale_pr_runs(api, source)
