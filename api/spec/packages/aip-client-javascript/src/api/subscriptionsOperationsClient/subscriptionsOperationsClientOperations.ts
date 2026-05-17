import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import {
  SubscriptionsOperationsClientContext,
} from "./subscriptionsOperationsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArraySubscriptionToApplicationTransform,
  jsonListSubscriptionsParamsFilterToTransportTransform,
  jsonPageMetaToApplicationTransform,
  jsonSortQueryToTransportTransform,
  jsonSubscriptionCancelToTransportTransform,
  jsonSubscriptionChangeResponseToApplicationTransform,
  jsonSubscriptionChangeToTransportTransform,
  jsonSubscriptionCreateToTransportTransform,
  jsonSubscriptionToApplicationTransform,
} from "../../models/internal/serializers.js";
import {
  type ListSubscriptionsParamsFilter,
  type SortQuery,
  type StringFieldFilterExact,
  Subscription,
  SubscriptionCancel,
  SubscriptionChange,
  type SubscriptionChangeResponse,
  type SubscriptionCreate,
  type UlidFieldFilter,
} from "../../models/models.js";

export interface CreateOptions extends OperationOptions {}
export async function create(
  client: SubscriptionsOperationsClientContext,
  subscription: SubscriptionCreate,
  options?: CreateOptions,
): Promise<Subscription | void> {
  const path = parse("/").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonSubscriptionCreateToTransportTransform(subscription),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonSubscriptionToApplicationTransform(response.body)!;
  }
  if (+response.status === 404 && !response.body) {
    return;
  }
  if (+response.status === 409 && !response.body) {
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
export interface ListOptions extends OperationOptions {
  size?: number
  number?: number
  page?: {
    /**
     * The number of items to include per page.
     */
    size?: number;
    /**
     * The page number.
     */
    number?: number;
  }
  sort?: SortQuery
  id?: UlidFieldFilter
  customerId?: UlidFieldFilter
  status?: StringFieldFilterExact
  planId?: UlidFieldFilter
  planKey?: StringFieldFilterExact
  filter?: ListSubscriptionsParamsFilter
}
export interface ListPageSettings {}
export interface ListPageResponse {
  data: Array<Subscription>
}
async function listSend(
  client: SubscriptionsOperationsClientContext,
  options?: Record<string, any>,
) {
  const path = parse("/{?page*,sort,filter*}").expand({
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }}),
    ...(options?.sort && {sort: jsonSortQueryToTransportTransform(options.sort)}),
    ...(options?.filter && {filter: jsonListSubscriptionsParamsFilterToTransportTransform(options.filter)})
  });
  const httpRequestOptions = {
    headers: {},
  };
  return await client.pathUnchecked(path).get(httpRequestOptions);;
}
function listDeserialize(
  response: PathUncheckedResponse,
  options?: ListOptions,
) {
  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return {
      data: jsonArraySubscriptionToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
    }!;
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
export function list(
  client: SubscriptionsOperationsClientContext,
  options?: ListOptions,
): PagedAsyncIterableIterator<Subscription,ListPageResponse,ListPageSettings> {
  function getElements(response: ListPageResponse) {
    return response.data;
  }
  async function getPagedResponse(
    nextToken?: string,
    settings?: ListPageSettings,
  ) {

            let response: PathUncheckedResponse;
            if (nextToken) {
              response = await client.pathUnchecked(nextToken).get();
            } else {
              const combinedOptions = { ...options, ...settings };
              response = await listSend(client, combinedOptions);
            }
    return {
    pagedResponse: await listDeserialize(response, options),
    nextToken: undefined,
    };
  }
  return buildPagedAsyncIterator<Subscription, ListPageResponse, ListPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface GetOptions extends OperationOptions {}
export async function get(
  client: SubscriptionsOperationsClientContext,
  subscriptionId: string,
  options?: GetOptions,
): Promise<Subscription | void> {
  const path = parse("/{subscriptionId}").expand({
    subscriptionId: subscriptionId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonSubscriptionToApplicationTransform(response.body)!;
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
export interface CancelOptions extends OperationOptions {}
/**
 * Cancels the subscription. Will result in a scheduling conflict if there are
 * other subscriptions scheduled to start after the cancelation time.
 *
 * @param {SubscriptionsOperationsClientContext} client
 * @param {string} subscriptionId
 * @param {SubscriptionCancel} body
 * @param {CancelOptions} [options]
 */
export async function cancel(
  client: SubscriptionsOperationsClientContext,
  subscriptionId: string,
  body: SubscriptionCancel,
  options?: CancelOptions,
): Promise<Subscription | void> {
  const path = parse("/{subscriptionId}/cancel").expand({
    subscriptionId: subscriptionId
  });
  const httpRequestOptions = {
    headers: {},body: jsonSubscriptionCancelToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonSubscriptionToApplicationTransform(response.body)!;
  }
  if (+response.status === 404 && !response.body) {
    return;
  }
  if (+response.status === 409 && !response.body) {
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
export interface UnscheduleCancelationOptions extends OperationOptions {}
/**
 * Unschedules the subscription cancelation.
 *
 * @param {SubscriptionsOperationsClientContext} client
 * @param {string} subscriptionId
 * @param {UnscheduleCancelationOptions} [options]
 */
export async function unscheduleCancelation(
  client: SubscriptionsOperationsClientContext,
  subscriptionId: string,
  options?: UnscheduleCancelationOptions,
): Promise<Subscription | void> {
  const path = parse("/{subscriptionId}/unschedule-cancelation").expand({
    subscriptionId: subscriptionId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonSubscriptionToApplicationTransform(response.body)!;
  }
  if (+response.status === 404 && !response.body) {
    return;
  }
  if (+response.status === 409 && !response.body) {
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
export interface ChangeOptions extends OperationOptions {}
/**
 * Closes a running subscription and starts a new one according to the
 * specification. Can be used for upgrades, downgrades, and plan changes.
 *
 * @param {SubscriptionsOperationsClientContext} client
 * @param {string} subscriptionId
 * @param {SubscriptionChange} body
 * @param {ChangeOptions} [options]
 */
export async function change(
  client: SubscriptionsOperationsClientContext,
  subscriptionId: string,
  body: SubscriptionChange,
  options?: ChangeOptions,
): Promise<SubscriptionChangeResponse | void> {
  const path = parse("/{subscriptionId}/change").expand({
    subscriptionId: subscriptionId
  });
  const httpRequestOptions = {
    headers: {},body: jsonSubscriptionChangeToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonSubscriptionChangeResponseToApplicationTransform(response.body)!;
  }
  if (+response.status === 404 && !response.body) {
    return;
  }
  if (+response.status === 409 && !response.body) {
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
