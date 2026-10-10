"""Cancel older PR runs by workflow run number, independent of queue arrival order."""

import json
import os
import time
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import HTTPRedirectHandler, Request, build_opener


class RejectRedirects(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        # Keep authenticated requests on the original GitHub API destination.
        raise HTTPError(req.full_url, code, "GitHub API redirects are disabled", headers, fp)


class GitHubAPI:
    def __init__(self, repository, token):
        # This controller runs on GitHub.com; alternative API hosts are unsupported.
        self.base_url = f"https://api.github.com/repos/{repository}"
        self.token = token
        self.opener = build_opener(RejectRedirects())

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
        # A cancellation whose response was lost can be retried: the controller
        # already handles the resulting 409 if GitHub accepted the first call.
        for attempt in range(3):
            try:
                with self.opener.open(request, timeout=30) as response:
                    body = response.read()
                    return json.loads(body) if body else None
            except HTTPError as error:
                # Handle HTTPError before URLError, its parent, so permission
                # failures and cancellation conflicts remain immediate.
                if attempt == 2 or not (error.code == 408 or 500 <= error.code < 600):
                    raise
                error.close()
            except (URLError, TimeoutError, ConnectionError):
                if attempt == 2:
                    raise

            delay = 2 ** attempt
            print(f"Temporary GitHub API failure; retrying in {delay}s (attempt {attempt + 2}/3)")
            time.sleep(delay)

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


def cancel_stale_pr_runs(api, source, pull):
    if source["event"] != "pull_request" or not source.get("head_repository"):
        return

    head_repository = source["head_repository"]
    if (
        not pull["head"]["repo"]
        or pull["head"]["repo"]["id"] != pull["base"]["repo"]["id"]
        or head_repository["id"] != pull["base"]["repo"]["id"]
        or source["head_branch"] != pull["head"]["ref"]
    ):
        print("Skipping cancellation: run is not from this repository's PR branch")
        return

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
        and any(candidate["number"] == pull["number"] for candidate in run["pull_requests"])
    ]
    # The calling run may not yet appear in the list endpoint. Include its
    # latest state so delayed cancellation jobs still preserve newer runs.
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


def main():
    with open(os.environ["GITHUB_EVENT_PATH"]) as event_file:
        event = json.load(event_file)
    pull = event.get("pull_request")
    if (
        not pull
        or not pull["head"]["repo"]
        or pull["head"]["repo"]["id"] != pull["base"]["repo"]["id"]
        or pull["user"]["login"] == "dependabot[bot]"
    ):
        print("Skipping cancellation: only internal, non-Dependabot PRs are supported")
        return

    api = GitHubAPI(
        os.environ["GITHUB_REPOSITORY"],
        os.environ["GITHUB_TOKEN"],
    )
    source = api.request(f"actions/runs/{os.environ['GITHUB_RUN_ID']}")
    cancel_stale_pr_runs(api, source, pull)


if __name__ == "__main__":
    main()
