import "./app.tsp";

namespace Apps;

/**
 * Sandbox app can be used for testing billing features.
 */
@friendlyName("BillingAppSandbox")
model AppSandbox {
  ...AppBase<AppType.Sandbox>;
}
