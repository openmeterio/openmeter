import "../shared/index.tsp";
import "./catalog.tsp";
import "./sandbox.tsp";
import "./stripe.tsp";
import "./external_invoicing.tsp";

namespace Apps;

/**
 * The type of the app.
 */
@friendlyName("BillingAppType")
enum AppType {
  /**
   * Built-in sandbox integration for testing and development.
   */
  Sandbox: "sandbox",

  /**
   * The Stripe app synchronizes invoices to Stripe Invoicing, enabling automated revenue collection with Stripe Payments and Stripe Tax.
   */
  Stripe: "stripe",

  /**
   * The external invoicing app enables synchronizing invoices with finance systems that are not natively supported, such as ERPs, in-house invoicing solutions, or local e-invoicing and payment providers.
   */
  ExternalInvoicing: "external_invoicing",
}

/**
 * Connection status of an installed app.
 */
@friendlyName("BillingAppStatus")
enum AppStatus {
  /**
   * The app is ready to be used.
   */
  Ready: "ready",

  /**
   * The app is unauthorized.
   * This usually happens when the app's credentials are revoked or expired.
   * To resolve this, the user must re-authorize the app.
   */
  Unauthorized: "unauthorized",
}

/**
 * Base model for installed apps, with its own configuration and credentials.
 */
@friendlyName("BillingAppBase")
model AppBase<T extends AppType> {
  ...Shared.Resource;

  /**
   * The app type.
   */
  // Update-visible so the discriminator survives visibility filtering in
  // UpdateRequestNested-derived request models; being a single-value enum per
  // variant, it only echoes the app's immutable type.
  @visibility(Lifecycle.Read, Lifecycle.Update)
  type: T;

  /**
   * The app catalog definition that this installed app is based on.
   */
  @visibility(Lifecycle.Read)
  definition: AppCatalogItem;

  /**
   * Status of the app connection.
   */
  @visibility(Lifecycle.Read)
  status: AppStatus;
}

/**
 * Installed application.
 */
@friendlyName("BillingApp")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union App {
  @summary("Stripe")
  stripe: AppStripe,

  @summary("Sandbox")
  sandbox: AppSandbox,

  @summary("External Invoicing")
  external_invoicing: AppExternalInvoicing,
}

/**
 * App reference.
 */
@friendlyName("BillingAppReference")
model AppReference {
  /**
   * The ID of the app.
   */
  id: Shared.ULID;
}

/**
 * Request to update an installed app.
 */
@friendlyName("BillingUpdateAppRequest")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union UpdateAppRequest {
  @summary("Stripe")
  stripe: Shared.UpdateRequestNested<AppStripe>,

  @summary("Sandbox")
  sandbox: Shared.UpdateRequestNested<AppSandbox>,

  @summary("External Invoicing")
  external_invoicing: Shared.UpdateRequestNested<AppExternalInvoicing>,
}
