namespace Shared;

/**
 * Identifies which system manages a resource.
 *
 * Values:
 * - `manual`: The resource is managed manually (overridden by our API users).
 * - `system`: The resource is managed by the system.
 * - `subscription`: The resource is managed by the subscription.
 */
@friendlyName("ResourceManagedBy")
@summary("Resource managed by")
enum ResourceManagedBy {
  Manual: "manual",
  System: "system",
  Subscription: "subscription",
}
