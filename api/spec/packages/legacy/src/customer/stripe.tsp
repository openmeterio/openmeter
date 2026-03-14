import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

import "../rest.tsp";
import "../errors.tsp";
import "../types.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter;
/**
 * Customer Stripe API.
 */
@route("/api/v1/customers/{customerIdOrKey}/stripe")
@tag("Customers")
@tag("Apps")
@friendlyName("CustomerStripe")
interface CustomerStripeEndpoints {
  /**
   * Get stripe app data for a customer.
   * Only returns data if the customer billing profile is linked to a stripe app.
   */
  @get
  @operationId("getCustomerStripeAppData")
  @summary("Get customer stripe app data")
  get(
    @path customerIdOrKey: ULIDOrExternalKey,
  ): StripeCustomerAppData | NotFoundError | CommonErrors;

  /**
   * Upsert stripe app data for a customer.
   * Only updates data if the customer billing profile is linked to a stripe app.
   */
  @put
  @operationId("upsertCustomerStripeAppData")
  @summary("Upsert customer stripe app data")
  upsert(
    @path customerIdOrKey: ULIDOrExternalKey,
    @body appData: StripeCustomerAppDataBase,
  ): StripeCustomerAppData | NotFoundError | CommonErrors;

  /**
   * Create Stripe customer portal session.
   * Only returns URL if the customer billing profile is linked to a stripe app and customer.
   *
   * Useful to redirect the customer to the Stripe customer portal to manage their payment methods,
   * change their billing address and access their invoice history.
   */
  @post
  @route("/portal")
  @operationId("createCustomerStripePortalSession")
  @summary("Create Stripe customer portal session")
  createPortalSession(
    @path customerIdOrKey: ULIDOrExternalKey,
    @body params: CreateStripeCustomerPortalSessionParams,
  ): {
    @statusCode _: 201;
    @body body: StripeCustomerPortalSession;
  } | NotFoundError | CommonErrors;
}

/**
 * Stripe customer portal request params.
 */
@friendlyName("CreateStripeCustomerPortalSessionParams")
model CreateStripeCustomerPortalSessionParams {
  /**
   * The ID of an existing configuration to use for this session,
   * describing its functionality and features.
   * If not specified, the session uses the default configuration.
   *
   * See https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-configuration
   */
  @summary("Configuration")
  configurationId?: string;

  /**
   * The IETF language tag of the locale customer portal is displayed in.
   * If blank or auto, the customer’s preferred_locales or browser’s locale is used.
   *
   * See: https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-locale
   */
  @summary("Locale")
  locale?: string;

  /**
   * The URL to redirect the customer to after they have completed
   * their requested actions.
   *
   * See: https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-return_url
   */
  @summary("ReturnUrl")
  returnUrl?: string;
}

/**
 * Stripe customer portal session.
 *
 * See: https://docs.stripe.com/api/customer_portal/sessions/object
 */
@friendlyName("StripeCustomerPortalSession")
model StripeCustomerPortalSession {
  /**
   * The ID of the customer portal session.
   *
   * See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-id
   */
  id: string;

  /**
   * The ID of the stripe customer.
   */
  stripeCustomerId: string;

  /**
   * Configuration used to customize the customer portal.
   *
   * See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-configuration
   */
  configurationId: string;

  /**
   * Livemode.
   *
   * See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-livemode
   */
  livemode: boolean;

  /**
   * Created at.
   *
   * See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-created
   */
  createdAt: DateTime;

  /**
   * Return URL.
   *
   * See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-return_url
   */
  returnUrl: string;

  /**
   * Status.
  /**
   * The IETF language tag of the locale customer portal is displayed in.
   *
   * See: https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-locale
   */
  locale: string;

  /**
  /**
   * The ID of the customer.
   */
  /**
   * The URL to redirect the customer to after they have completed
   * their requested actions.
   */
  url: string;
}
