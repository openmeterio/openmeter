import { parse } from "uri-template";
import {
  CustomerBillingOperationsClientContext,
} from "./customerBillingOperationsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  jsonAppCustomerDataToApplicationTransform,
  jsonCreateStripeCheckoutSessionResultToApplicationTransform,
  jsonCreateStripeCustomerPortalSessionResultToApplicationTransform,
  jsonCustomerBillingDataToApplicationTransform,
  jsonCustomerBillingStripeCreateCheckoutSessionRequestToTransportTransform,
  jsonCustomerBillingStripeCreateCustomerPortalSessionRequestToTransportTransform,
  jsonUpsertRequestToTransportTransform_2,
  jsonUpsertRequestToTransportTransform_3,
} from "../../models/internal/serializers.js";
import {
  type AppCustomerData,
  type CreateStripeCheckoutSessionResult,
  type CreateStripeCustomerPortalSessionResult,
  type CustomerBillingData,
  CustomerBillingStripeCreateCheckoutSessionRequest,
  CustomerBillingStripeCreateCustomerPortalSessionRequest,
  type UpsertRequest_2 as UpsertRequest,
  type UpsertRequest_3 as UpsertRequest_2,
} from "../../models/models.js";

export interface GetOptions extends OperationOptions {}
export async function get(
  client: CustomerBillingOperationsClientContext,
  customerId: string,
  options?: GetOptions,
): Promise<CustomerBillingData | void> {
  const path = parse("/{customerId}").expand({
    customerId: customerId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCustomerBillingDataToApplicationTransform(response.body)!;
  }
  if (+response.status === 404 && !response.body) {
    return;
  }
  if (+response.status === 400 && !response.body) {
    return;
  }
  if (+response.status === 401 && !response.body) {
    return;
  }
  if (+response.status === 403 && !response.body) {
    return;
  }
  throw createRestError(response);
}
;
export interface UpsertOptions extends OperationOptions {}
export async function upsert(
  client: CustomerBillingOperationsClientContext,
  customerId: string,
  body: UpsertRequest,
  options?: UpsertOptions,
): Promise<CustomerBillingData | void> {
  const path = parse("/{customerId}").expand({
    customerId: customerId
  });
  const httpRequestOptions = {
    headers: {},body: jsonUpsertRequestToTransportTransform_2(body),
  };
  const response = await client.pathUnchecked(path).put(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCustomerBillingDataToApplicationTransform(response.body)!;
  }
  if (+response.status === 410 && !response.body) {
    return;
  }
  if (+response.status === 404 && !response.body) {
    return;
  }
  if (+response.status === 400 && !response.body) {
    return;
  }
  if (+response.status === 401 && !response.body) {
    return;
  }
  if (+response.status === 403 && !response.body) {
    return;
  }
  throw createRestError(response);
}
;
export interface UpsertAppDataOptions extends OperationOptions {}
export async function upsertAppData(
  client: CustomerBillingOperationsClientContext,
  customerId: string,
  body: UpsertRequest_2,
  options?: UpsertAppDataOptions,
): Promise<AppCustomerData | void> {
  const path = parse("/app-data/{customerId}").expand({
    customerId: customerId
  });
  const httpRequestOptions = {
    headers: {},body: jsonUpsertRequestToTransportTransform_3(body),
  };
  const response = await client.pathUnchecked(path).put(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonAppCustomerDataToApplicationTransform(response.body)!;
  }
  if (+response.status === 410 && !response.body) {
    return;
  }
  if (+response.status === 404 && !response.body) {
    return;
  }
  if (+response.status === 400 && !response.body) {
    return;
  }
  if (+response.status === 401 && !response.body) {
    return;
  }
  if (+response.status === 403 && !response.body) {
    return;
  }
  throw createRestError(response);
}
;
export interface CreateCheckoutSessionOptions extends OperationOptions {}
/**
 * Create a [Stripe Checkout Session](https://docs.stripe.com/payments/checkout)
 * for the customer. Creates a Checkout Session for collecting payment method
 * information from customers. The session operates in "setup" mode, which
 * collects payment details without charging the customer immediately. The
 * collected payment method can be used for future subscription billing. For
 * hosted checkout sessions, redirect customers to the returned URL. For
 * embedded sessions, use the client_secret to initialize Stripe.js in your
 * application.
 *
 * @param {CustomerBillingOperationsClientContext} client
 * @param {string} customerId
 * @param {CustomerBillingStripeCreateCheckoutSessionRequest} body
 * @param {CreateCheckoutSessionOptions} [options]
 */
export async function createCheckoutSession(
  client: CustomerBillingOperationsClientContext,
  customerId: string,
  body: CustomerBillingStripeCreateCheckoutSessionRequest,
  options?: CreateCheckoutSessionOptions,
): Promise<CreateStripeCheckoutSessionResult | void> {
  const path = parse("/stripe/checkout-sessions/{customerId}").expand({
    customerId: customerId
  });
  const httpRequestOptions = {
    headers: {

    },body: jsonCustomerBillingStripeCreateCheckoutSessionRequestToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCreateStripeCheckoutSessionResultToApplicationTransform(response.body)!;
  }
  if (+response.status === 410 && !response.body) {
    return;
  }
  if (+response.status === 404 && !response.body) {
    return;
  }
  if (+response.status === 400 && !response.body) {
    return;
  }
  if (+response.status === 401 && !response.body) {
    return;
  }
  if (+response.status === 403 && !response.body) {
    return;
  }
  throw createRestError(response);
}
;
export interface CreatePortalSessionOptions extends OperationOptions {}
/**
 * Create Stripe Customer Portal Session. Useful to redirect the customer to the
 * Stripe Customer Portal to manage their payment methods, change their billing
 * address and access their invoice history. Only returns URL if the customer
 * billing profile is linked to a stripe app and customer.
 *
 * @param {CustomerBillingOperationsClientContext} client
 * @param {string} customerId
 * @param {CustomerBillingStripeCreateCustomerPortalSessionRequest} body
 * @param {CreatePortalSessionOptions} [options]
 */
export async function createPortalSession(
  client: CustomerBillingOperationsClientContext,
  customerId: string,
  body: CustomerBillingStripeCreateCustomerPortalSessionRequest,
  options?: CreatePortalSessionOptions,
): Promise<CreateStripeCustomerPortalSessionResult | void> {
  const path = parse("/stripe/portal-sessions/{customerId}").expand({
    customerId: customerId
  });
  const httpRequestOptions = {
    headers: {

    },body: jsonCustomerBillingStripeCreateCustomerPortalSessionRequestToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCreateStripeCustomerPortalSessionResultToApplicationTransform(response.body)!;
  }
  if (+response.status === 410 && !response.body) {
    return;
  }
  if (+response.status === 404 && !response.body) {
    return;
  }
  if (+response.status === 400 && !response.body) {
    return;
  }
  if (+response.status === 401 && !response.body) {
    return;
  }
  if (+response.status === 403 && !response.body) {
    return;
  }
  throw createRestError(response);
}
;
