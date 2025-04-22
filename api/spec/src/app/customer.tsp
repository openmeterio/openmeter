import "./app.tsp";
import "../types.tsp";

using OpenMeter.App;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.App;

/**
 * CustomerAppData
 * Stores the app specific data for the customer.
 * One of: stripe, sandbox, custom_invoicing
 */
@friendlyName("CustomerAppData")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union CustomerAppData {
  @summary("Stripe")
  stripe: StripeCustomerAppData,

  @summary("Sandbox")
  sandbox: SandboxCustomerAppData,

  @summary("Custom Invoicing")
  custom_invoicing: CustomInvoicingCustomerAppData,
}

/**
 * Customer App Base.
 */
@friendlyName("CustomerAppDataBase")
model CustomerAppBase<TYPE extends Type> {
  /**
   * The app ID.
   * If not provided, it will use the global default for the app type.
   */
  @summary("App ID")
  id?: ULID;

  /**
   * The app name.
   */
  @summary("App Type")
  type: TYPE;
}

/**
 * Sandbox Customer App Data.
 */
@friendlyName("SandboxCustomerAppData")
model SandboxCustomerAppData {
  /**
   * The installed sandbox app this data belongs to.
   */
  @visibility(Lifecycle.Read)
  app?: SandboxApp;

  ...CustomerAppBase<Type.Sandbox>;
}

/**
 * Stripe Customer App Data.
 */
@example(#{ type: Type.Stripe, stripeCustomerId: "cus_xxxxxxxxxxxxxx" })
@friendlyName("StripeCustomerAppData")
model StripeCustomerAppData {
  ...CustomerAppBase<Type.Stripe>;

  /**
   * The installed stripe app this data belongs to.
   */
  @visibility(Lifecycle.Read)
  app?: StripeApp;

  /**
   * The Stripe customer ID.
   */
  stripeCustomerId: string;

  /**
   * The Stripe default payment method ID.
   */
  stripeDefaultPaymentMethodId?: string;
}

/**
 * Custom Invoicing Customer App Data.
 */
@friendlyName("CustomInvoicingCustomerAppData")
model CustomInvoicingCustomerAppData {
  /**
   * The installed custom invoicing app this data belongs to.
   */
  @visibility(Lifecycle.Read)
  app?: CustomInvoicingApp;

  ...CustomerAppBase<Type.CustomInvoicing>;

  /**
   * Metadata to be used by the custom invoicing provider.
   */
  metadata?: Metadata;
}
