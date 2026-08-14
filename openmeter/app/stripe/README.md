# Stripe app

Stripe credentials are app-owned secrets. A Stripe app can only persist API-key and webhook-secret
references whose app ID, namespace, and semantic key match the app being created. Secret reads and
updates preserve that ownership rather than allowing callers to combine the namespace, app ID, or
key from different references.

App persistence permits only two credential states: active rows retain both the API-key and
webhook-secret references, while soft-deleted rows retain neither. A deletion transition must set
`deleted_at` and clear both references atomically; partial credential removal is invalid.

This lifecycle makes the nullable-secret migration conditionally irreversible. After a
soft-deleted row exists, its migration cannot be rolled back to `NOT NULL` columns until that row is
purged or both valid secret references are reconstructed. Purging soft-deleted Stripe subtype rows
is the expected rollback preparation because uninstallation intentionally removes the underlying
secrets.

## Webhook namespace bootstrap

Stripe webhook routes receive a globally unique app ID but no trusted namespace. Resolving the
webhook secret is therefore the deliberate exception to the normal namespace-scoped app lookup:
the adapter resolves the Stripe app by ID, derives its namespace from that row, and uses the pair to
load the app-owned signing secret.

The derived namespace becomes trusted request context only after the Stripe signature has been
verified with that secret. Namespace-free app lookup must not be reused for ordinary authenticated
app operations, where the caller's namespace is already known and must constrain every query.
