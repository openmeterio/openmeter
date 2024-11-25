import "./app.tsp";
import "../types.tsp";

using OpenMeter.App;

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.App;

/**
 * CustomerApp
 * Stores the app specific data for the customer.
 * One of: stripe
 */
@friendlyName("CustomerApp")
@discriminator("type")
union CustomerApp {
  @summary("Stripe")
  stripe: StripeCustomerApp,
}

/**
 * Customer App Base.
 */
@example(#{
  type: Type.Stripe,
  data: #{ stripeCustomerId: "cus_xxxxxxxxxxxxxx" },
})
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
 * Stripe Customer App.
 */
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
