import "./app.tsp";
import "../types.tsp";

using OpenMeter.App;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.App;

/**
 * CustomerApp
 * Stores the app specific data for the customer.
 * One of: stripe, sandbox
 */
@friendlyName("CustomerApp")
@discriminator("type")
union CustomerApp {
  @summary("Stripe")
  stripe: StripeCustomerApp,

  @summary("Sandbox")
  sandbox: SandboxCustomerApp,
}

/**
 * Customer App Base.
 */
@friendlyName("CustomerAppBase")
model CustomerAppBase<TYPE extends Type, DATA> {
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

  /**
   * The app data.
   */
  @summary("Data")
  data: DATA;
}

/**
 * Sandbox Customer App.
 */
@friendlyName("SandboxCustomerApp")
model SandboxCustomerApp {
  ...CustomerAppBase<Type.Sandbox, SandboxCustomerAppData>;
}

/**
 * Sandbox Customer App Data.
 */
@friendlyName("SandboxCustomerAppData")
model SandboxCustomerAppData {}

/**
 * Stripe Customer App.
 */
@example(#{
  type: Type.Stripe,
  data: #{ stripeCustomerId: "cus_xxxxxxxxxxxxxx" },
})
@friendlyName("StripeCustomerApp")
model StripeCustomerApp {
  ...CustomerAppBase<Type.Stripe, StripeCustomerAppData>;
}

/**
 * Stripe Customer App Data.
 */
@example(#{ stripeCustomerId: "cus_xxxxxxxxxxxxxx" })
@friendlyName("StripeCustomerAppData")
model StripeCustomerAppData {
  /**
   * The Stripe customer ID.
   */
  stripeCustomerId: string;

  /**
   * The Stripe default payment method ID.
   */
  stripeDefaultPaymentMethodId?: string;
}
