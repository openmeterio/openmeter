# Notification implementation follow-ups

- **Svix’s provider-managed routing needs additional care.** Excluding a Channel
  from Notification’s destination list does not prevent Svix from delivering to
  its endpoint: Svix routes using endpoint configuration. Before activating
  asynchronous Channel reconciliation, define how to delay affected sends while
  endpoint routing is unconfirmed or outdated (`unknown`, `pending`, or `error`).
  Determine the required coordination scope, potentially the namespace/application,
  and how dispatch resumes after routing converges. Cover stale subscriptions,
  filter removal, disabled/deleted endpoints, and partially completed provisioning.
  Continue observing deliveries already accepted by the provider while new
  dispatches wait; do not silently drop waiting destinations.
