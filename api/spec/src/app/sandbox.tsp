namespace OpenMeter;

/**
 * Sandbox app can be used for testing OpenMeter features.
 *
 * The app is not creating anything in external systems, thus it is safe to use for
 * verifying OpenMeter features.
 */
@friendlyName("SandboxApp")
model SandboxApp {
  ...AppBase;

  /**
   * The app's type is Sandbox.
   */
  type: AppType.Sandbox;
}
