namespace Shared;

/**
 * Identifies whether a resource lifecycle is controlled by OpenMeter or manually
 * overridden by the API user.
 *
 * Values:
 *
 * - `system`: The resource lifecycle is controlled by OpenMeter.
 * - `manual`: The resource lifecycle was manually overridden by the API user.
 */
@friendlyName("BillingLifecycleController")
@summary("Lifecycle controller")
enum LifecycleController {
  System: "system",
  Manual: "manual",
}
