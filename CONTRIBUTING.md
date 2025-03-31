# Contributing

Thanks for your interest in contributing to OpenMeter!

Here are a few general guidelines on contributing and reporting bugs that we ask you to review.
Following these guidelines helps to communicate that you respect the time of the contributors managing and developing this open source project.
In return, they should reciprocate that respect in addressing your issue, assessing changes, and helping you finalize your pull requests.
In that spirit of mutual respect, we endeavor to review incoming issues and pull requests within 10 days,
and will close any lingering issues or pull requests after 60 days of inactivity.

Please note that all of your interactions in the project are subject to our [Code of Conduct](/CODE_OF_CONDUCT.md).
This includes creation of issues or pull requests, commenting on issues or pull requests,
and extends to all interactions in any real-time space e.g., Slack, Discord, etc.

## Reporting issues

Before reporting a new issue, please ensure that the issue was not already reported or fixed by searching through our issue tracker.

When creating a new issue, please be sure to include a **title and clear description**, as much relevant information as possible, and, if possible, a test case.

**If you discover a security bug, please do not report it through GitHub issues. Instead, please see security procedures in [SECURITY.md](/SECURITY.md).**

## Sending pull requests

Before sending a new pull request, take a look at existing pull requests and issues to see if the proposed change or fix has been discussed in the past,
or if the change was already implemented but not yet released.

We expect new pull requests to include tests for any affected behavior, and, as we follow semantic versioning,
we may reserve breaking changes until the next major version release.

### Ensuring All Requested Reviewers Approve

By default, pull requests can often be merged once the minimum number of required approvals (e.g., from CODEOWNERS or branch protection rules) is met. However, sometimes you might explicitly request reviews from specific individuals because their input is crucial for that particular PR.

To ensure that *all* individuals you've specifically requested using the GitHub "Reviewers" UI must approve before merging, follow these steps:

1.  **Request Reviews:** Use the standard GitHub interface on the pull request page to request reviews from the necessary individuals.
2.  **Add Label:** Add the label `require-all-reviewers` to the pull request.

When this label is present, an automated check named "Review Gatekeeper" will run. This check will only pass if **every single user** listed under the "Reviewers" section has submitted an **approving** review. This check is required for merging, preventing merges until all explicitly requested reviewers are satisfied.

If the label is removed, the "Review Gatekeeper" check will be skipped.

## Other ways to contribute

We welcome anyone that wants to contribute to triage and reply to open issues to help troubleshoot and fix existing bugs.
Here is what you can do:

- Help ensure that existing issues follows the recommendations from the _[Reporting Issues](#reporting-issues)_ section,
  providing feedback to the issue's author on what might be missing.
- Review and update the existing content of our [documentation](https://openmeter.io) with up-to-date instructions and code samples.
- Review existing pull requests, and testing patches against real existing applications.
- Write a test, or add a missing test case to an existing test.
