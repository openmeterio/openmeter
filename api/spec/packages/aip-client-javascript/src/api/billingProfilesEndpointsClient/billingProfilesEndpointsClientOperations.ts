import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import {
  BillingProfilesEndpointsClientContext,
} from "./billingProfilesEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayBillingProfileToApplicationTransform,
  jsonBillingProfileToApplicationTransform,
  jsonCreateRequestToTransportTransform_4,
  jsonPageMetaToApplicationTransform,
  jsonUpsertRequestToTransportTransform_4,
} from "../../models/internal/serializers.js";
import {
  BillingProfile,
  CreateRequest_4 as CreateRequest,
  UpsertRequest_4 as UpsertRequest,
} from "../../models/models.js";

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
}
export interface ListPageSettings {}
export interface ListPageResponse {
  data: Array<BillingProfile>
}
async function listSend(
  client: BillingProfilesEndpointsClientContext,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/profiles{?page*}").expand({
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }})
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
      data: jsonArrayBillingProfileToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
    }!;
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
  client: BillingProfilesEndpointsClientContext,
  options?: ListOptions,
): PagedAsyncIterableIterator<BillingProfile,ListPageResponse,ListPageSettings> {
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
  return buildPagedAsyncIterator<BillingProfile, ListPageResponse, ListPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface CreateOptions extends OperationOptions {}
/**
 * Create a new billing profile. Billing profiles contain the settings for
 * billing and controls invoice generation. An organization can have multiple
 * billing profiles defined. A billing profile is linked to a specific app. This
 * association is established during the billing profile's creation and remains
 * immutable.
 *
 * @param {BillingProfilesEndpointsClientContext} client
 * @param {CreateRequest} profile
 * @param {CreateOptions} [options]
 */
export async function create(
  client: BillingProfilesEndpointsClientContext,
  profile: CreateRequest,
  options?: CreateOptions,
): Promise<BillingProfile | void> {
  const path = parse("/openmeter/profiles").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonCreateRequestToTransportTransform_4(profile),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonBillingProfileToApplicationTransform(response.body)!;
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
export interface GetOptions extends OperationOptions {}
/**
 * Get a billing profile.
 *
 * @param {BillingProfilesEndpointsClientContext} client
 * @param {string} id
 * @param {GetOptions} [options]
 */
export async function get(
  client: BillingProfilesEndpointsClientContext,
  id: string,
  options?: GetOptions,
): Promise<BillingProfile | void> {
  const path = parse("/openmeter/profiles/{id}").expand({
    id: id
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonBillingProfileToApplicationTransform(response.body)!;
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
export interface UpdateOptions extends OperationOptions {}
/**
 * Update a billing profile.
 *
 * @param {BillingProfilesEndpointsClientContext} client
 * @param {string} id
 * @param {UpsertRequest} profile
 * @param {UpdateOptions} [options]
 */
export async function update(
  client: BillingProfilesEndpointsClientContext,
  id: string,
  profile: UpsertRequest,
  options?: UpdateOptions,
): Promise<BillingProfile | void> {
  const path = parse("/openmeter/profiles/{id}").expand({
    id: id
  });
  const httpRequestOptions = {
    headers: {},body: jsonUpsertRequestToTransportTransform_4(profile),
  };
  const response = await client.pathUnchecked(path).put(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonBillingProfileToApplicationTransform(response.body)!;
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
export interface DeleteOptions extends OperationOptions {}
/**
 * Delete a billing profile. Only such billing profiles can be deleted that are:
 * - not the default profile - not pinned to any customer using customer
 * overrides - only have finalized invoices
 *
 * @param {BillingProfilesEndpointsClientContext} client
 * @param {string} id
 * @param {DeleteOptions} [options]
 */
export async function delete_(
  client: BillingProfilesEndpointsClientContext,
  id: string,
  options?: DeleteOptions,
): Promise<void> {
  const path = parse("/openmeter/profiles/{id}").expand({
    id: id
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).delete(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 204 && !response.body) {
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
