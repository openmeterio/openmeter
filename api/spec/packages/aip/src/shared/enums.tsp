namespace Shared;

/**
 * Identifies which system manages a resource.
 */
@friendlyName("ResourceManagedBy")
@summary("Resource managed by")
enum ResourceManagedBy {
  /**
   * The resource is managed manually (overridden by our API users).
   */
  Manual: "manual",

  /**
   * The resource is managed by the system.
   */
  System: "system",

  /**
   * The resource is managed by the subscription.
   */
  Subscription: "subscription",
}
