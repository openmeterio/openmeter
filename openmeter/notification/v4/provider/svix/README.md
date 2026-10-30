# Svix provider

A Notification namespace is a Svix application. Its Notification ID is the
application UID; the generated application ID is returned separately as
`ProviderData["svix.application_id"]`. User metadata never carries provider data.
Notification owns persistence of these results.

Creation uses UID-based get-or-create, so a retry returns the existing
application without overwriting its configuration. Updates require an existing
application and change only supplied fields. Provided metadata replaces the
complete map; a pointer to a nil or empty map clears old keys, while an omitted
pointer preserves them. Updates leave UID and unrelated settings such as throttling
untouched. Deleting an absent application succeeds.

## Channels

A Channel maps to an endpoint in its namespace's application. Its Channel ID is
the endpoint UID; `ProviderData["svix.endpoint_id"]` carries the generated Svix
ID. Namespace provisioning is separate: Channel operations do not create an
application implicitly. Notification remains responsible for local soft deletion.

Notification owns typed `v4.WebhookConfig`, `v4.WebhookConfigUpdate`, and their
URL/signing-secret validation. The registry binds Svix to
`Provider[WebhookConfig, WebhookConfigUpdate]`, without runtime type assertions
or JSON decoding. Creation requires a complete configuration; updates validate
and apply only explicitly supplied fields.
Its serialized fields remain `url`, optional `custom_headers`, and optional
`signing_secret`. Provided headers replace the complete map; a pointer to a nil
or empty map clears them. Omitted headers are preserved. Omitting the secret lets
Svix generate it on creation and preserves it on update. An explicitly empty
update secret is invalid, not a request to generate another key.
An explicit secret is compared before rotation to make retries safe. Secrets are
never returned in ProviderData. Encryption and public-response redaction are
not implemented by these configuration types.

The Channel name becomes the endpoint description. Routing uses canonical
Event types supplied by Notification; the provider neither expands wildcards
nor registers Event types. Nil or empty `EventTypes` on creation means catch-all
within the application. On update, an omitted pointer preserves filtering;
a pointer to a nil or empty slice selects catch-all. The provider honors
`Disabled` independently of filtering; an omitted update pointer preserves it.
Notification must supply an effective disabled state when a nonempty user filter
resolves to no types, without changing the user's desired disabled state.
Updates clear filters using explicit `eventTypes: null` (Svix rejects `[]` on
PATCH) and also clear legacy Rule-ID channel filters in the same request.
Unrelated updates leave both routing filters unchanged.

Create retries recover UID conflicts by applying the desired configuration to
the existing endpoint. Creation explicitly sets headers after creating the
endpoint because some Svix servers ignore headers in the create request; a
header failure leaves the endpoint for a retry and does not report success.
Updates require an existing endpoint, and report success only after all supplied
changes succeed. Empty updates only read the existing resource. Header/secret-only
updates verify existence without patching endpoint fields. A retry reapplies the
same supplied changes after partial failure. `in_sync` confirms those changes,
not the state of omitted fields; full reconciliation must explicitly supply all
managed fields. Deleting an absent endpoint
succeeds. Get only confirms that provisioning exists; its input contains no
desired configuration, so it cannot detect configuration drift.

All provider operations preserve cancellation and deadline errors through
contextual wrapping without classifying them as `ProviderError`. Other SDK
failures are normalized for reconciliation. Cancellation during a multi-step
Channel mutation returns no success result, but earlier remote changes may
already have taken effect; a retry must reconcile the desired state.
The pinned SDK does not expose response headers on errors, so
retryable errors leave `RetryAfter` unset for the reconciler's default backoff.

## Events

Initial delivery creates one Svix message with the Notification Event ID as its
external identity and the canonical versioned Event type. Endpoint configuration
owns fan-out; the supplied destinations describe Notification's tracking set,
not explicit routing. A duplicate Event ID is recovered by reading the existing
message. Acceptance is not delivery success.

Application composition supplies Svix through `provider.RegistryConfig.Webhook`.
`provider.NewRegistry` rejects missing dependencies before exposing providers
through accessors such as `Webhook()`. Consumers must use a constructed registry,
not its zero value. `RoutingMode()` reports `event_type` for initial sends;
targeted resends are unaffected by that mode.

The webhook envelope contains `id`, `event_type`, `created_at`, raw JSON
`payload`, `annotations`, and an optional root-level `rule_snapshot`. Provider
data and Rule IDs are not exposed. No Registry or domain lookup is performed.

Resend targets each requested endpoint independently and returns partial
outcomes. Successful `GetEvent` observations return sorted attempt IDs in
`svix.observed_attempts`, alongside `svix.dispatch_generation`, for the reconciler
to persist during normal operation. For a new resend generation, both
`ResendEvent` and `GetEvent` use the previous generation's persisted observation
as `svix.resend_baseline`. For the same generation, they retain its established
baseline rather than rebuilding it from current remote attempts.

```text
Persisted: generation=1, observed_attempts=[A]
Resend generation=2: Svix accepts attempt B, but the response is lost
GetEvent(generation=2, persisted generation=1 data) returns:
  generation=2, observed_attempts=[A,B], resend_baseline=[A]
```

Observation retains all attempts but excludes the baseline when deciding the
current generation's state. No new attempt means `sending`, not success or a
reason to resend automatically. Repeated and out-of-band observations refresh
the snapshot without changing that generation's baseline; the next explicit
resend uses the latest persisted snapshot. Retry an unresolved resend with the
same generation, persisted input, and caller-supplied idempotency key.

Missing, malformed, or mismatched observation data blocks resend with a
configuration error; observation reports a retryable error. An explicit empty
JSON array is valid, but missing or `null` snapshots are not. Neither operation
reconstructs a missing baseline from live attempts. Reconciliation owns
persistence, synchronization, scheduling, and delivery timeouts; no provider
cache, persistence callback, or extra checkpoint operation is introduced.
This baseline does not correlate overlapping resends made outside Notification.

Observation paginates destinations and attempts. Individual endpoint failures
do not discard other observations. Failed attempts with a scheduled next attempt
remain `sending`; canceled attempts map to `failed`. Message and attempt IDs are
returned in their respective ProviderData objects. Attempt response JSON is
limited to 64 KiB; oversized bodies are omitted with `response_truncated` inside
the response object.

Event operation and destination errors wrap their causes with resource context.
Callers use `errors.As` to inspect `ProviderError` classification and `errors.Is`
to detect cancellation or deadline expiry.
Cancellation during per-destination work preserves accumulated results and is
reported on the affected destination, with a nil method-level error. An
interrupted send has an uncertain outcome, not a confirmed rejection.

Event operation spans report method-level failures. Get and resend also create
destination spans, so partial errors remain visible without discarding results.
An OK parent span means the operation returned its results, not that all
destinations succeeded; `notification.destination.error_count` captures that
distinction. Observing a failed delivery successfully is not a tracing error.

Namespace creation, Channel creation, send, and resend inputs accept an optional
`IdempotencyKey`. A supplied key must be nonblank and is passed unchanged to
Svix, including each destination's resend call. Nil leaves key generation to
the SDK; the provider does not generate or derive keys. Event ID remains the
message's external identity independently of its idempotency key.

The SDK reuses the key during its built-in retries; no provider-level retry
loop is added. SDK-generated keys do not provide deduplication across separate
provider calls; callers must supply a stable key when that is required, subject
to Svix's idempotency window. Updates do not accept an explicit key,
including secret rotation. Keys are operation inputs, not resource fields or
ProviderData.
