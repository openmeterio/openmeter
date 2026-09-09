# Emergency production fix approval

An active member of `@openmeterio/openmeter-oncall` can apply `urgent-prod-fix`
to request a bot approval for an internal, non-draft PR targeting `main`. The
PR author must be an active OpenMeter organization member and the source branch
must belong to `openmeterio/openmeter`. Membership lookup failures deny approval.

The bot approves the revision present when the label was applied. It never merges
or enables auto-merge. Existing required CI checks continue to apply. After an
engineer merges a PR with an emergency approval for that revision, the workflow
posts its PR, merge commit, and authorization review links to Slack without
mentions. Existing auto-merge, if enabled by a user, can still react to an approval.

## Setup

1. Create an organization-owned GitHub App for emergency approvals. Disable its
   webhook; GitHub Actions handles events. Grant repository **Pull requests:
   read and write** and organization **Members: read-only** permissions. Install
   it on `openmeterio/openmeter` only. No branch-protection bypass is needed.
2. Add its client ID as the repository Actions variable `URGENT_FIX_APP_CLIENT_ID`.
3. Generate an App private key and store the complete PEM as the repository
   Actions secret `URGENT_FIX_APP_PRIVATE_KEY`.
4. Configure `URGENT_FIX_SLACK_WEBHOOK_URL` as a repository Actions secret.
5. Create the `urgent-prod-fix` PR label and manage authorized users in the
   `openmeter-oncall` team. Team membership authorizes requests regardless of
   who is currently on a PagerDuty shift; there is no schedule synchronization.
6. Keep **Dismiss stale pull request approvals when new commits are pushed**
   enabled on `main`. If review dismissal is restricted, allow the App to dismiss
   its emergency reviews. Keep the existing required status checks.

The privileged workflow checks out `github.workflow_sha`, never the PR head.
Tests run separately without the App credentials, only for PRs changing `.github/`.
Protect access to the App key
and changes to the workflow and helper: they are part of the authorization boundary.

## Operation and limits

Pushing to an open PR automatically removes `urgent-prod-fix`, preserving other
labels. Wait for the push workflow to finish, then reapply the label to request
approval for the new revision. After a failed request, remove and reapply it.
Re-running a labeled-event job cannot issue a new approval.
The workflow dismisses its own invalid approvals on pushes, label removal, or
conversion to draft; it does not dismiss human reviews. GitHub's stale-review
protection provides immediate invalidation for code-changing pushes.

Label removal and draft changes are processed asynchronously; they are not an
atomic merge lock. Membership is checked when approval is requested, not
continuously afterward. Removing someone from the team does not revoke an
already-submitted approval; dismiss that review explicitly if needed.

Slack notification failures fail the closed-event job. Rerun that job to retry;
a retry after an ambiguous network failure can produce a duplicate message.
Notification uses the matching bot review, so removing a label after merge does
not suppress the post-merge notification. This does not track review completion.

Before rollout, validate App approval counting, review dismissal, and Slack
delivery on an internal test PR with the actual installation. Do not merge until
the checks pass and a human is ready to review the change.

## Local validation

```sh
python3 -B -m unittest discover -s .github/tools/urgent-prod-fix -p 'test_*.py'
actionlint .github/workflows/urgent-prod-fix.yaml .github/workflows/urgent-prod-fix-tests.yaml
```
