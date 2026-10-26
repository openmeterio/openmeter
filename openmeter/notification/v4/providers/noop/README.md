# No-op webhook provider

`NewWebhookProvider(logger)` supplies the required registry entry when webhook
delivery is disabled. It makes no network calls, stores no resources, and never
returns fabricated provisioning, acceptance, or delivery success. Application
composition must explicitly select it; it is not a fallback for Svix failures.

Following the existing notification no-op webhook handler, valid operations
return a not-implemented cause, classified as `configuration` /
`provider_configuration_invalid`. This includes deletion: a disabled provider
cannot confirm that another provider's remote resources were removed. Results
are nil on error; no partial destination results are returned.

Inputs are validated, cancellation/deadline errors retain their original cause
without becoming ProviderErrors, and logs never include configuration or payloads.
Routing mode is `explicit`, but no routing or sending is performed.
