import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import {
  MetersEndpointsClientContext,
} from "./metersEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayMeterToApplicationTransform,
  jsonCreateRequestToTransportTransform,
  jsonListMetersParamsFilterToTransportTransform,
  jsonMeterToApplicationTransform,
  jsonPageMetaToApplicationTransform,
  jsonSortQueryToTransportTransform,
  jsonUpdateRequestToTransportTransform,
} from "../../models/internal/serializers.js";
import {
  CreateRequest,
  type ListMetersParamsFilter,
  Meter,
  type SortQuery,
  type StringFieldFilter,
  UpdateRequest,
} from "../../models/models.js";

export interface CreateOptions extends OperationOptions {}
/**
 * Create a meter.
 *
 * @param {MetersEndpointsClientContext} client
 * @param {CreateRequest} meter
 * @param {CreateOptions} [options]
 */
export async function create(
  client: MetersEndpointsClientContext,
  meter: CreateRequest,
  options?: CreateOptions,
): Promise<Meter | void> {
  const path = parse("/openmeter/meters").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonCreateRequestToTransportTransform(meter),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonMeterToApplicationTransform(response.body)!;
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
 * Get a meter by ID.
 *
 * @param {MetersEndpointsClientContext} client
 * @param {string} meterId
 * @param {GetOptions} [options]
 */
export async function get(
  client: MetersEndpointsClientContext,
  meterId: string,
  options?: GetOptions,
): Promise<Meter | void> {
  const path = parse("/openmeter/meters/{meterId}").expand({
    meterId: meterId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonMeterToApplicationTransform(response.body)!;
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
  key?: StringFieldFilter
  name?: StringFieldFilter
  filter?: ListMetersParamsFilter
}
export interface ListPageSettings {}
export interface ListPageResponse {
  data: Array<Meter>
}
async function listSend(
  client: MetersEndpointsClientContext,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/meters{?page*,sort,filter*}").expand({
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }}),
    ...(options?.sort && {sort: jsonSortQueryToTransportTransform(options.sort)}),
    ...(options?.filter && {filter: jsonListMetersParamsFilterToTransportTransform(options.filter)})
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
      data: jsonArrayMeterToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
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
  client: MetersEndpointsClientContext,
  options?: ListOptions,
): PagedAsyncIterableIterator<Meter,ListPageResponse,ListPageSettings> {
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
  return buildPagedAsyncIterator<Meter, ListPageResponse, ListPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface UpdateOptions extends OperationOptions {}
/**
 * Update a meter.
 *
 * @param {MetersEndpointsClientContext} client
 * @param {string} meterId
 * @param {UpdateRequest} meter
 * @param {UpdateOptions} [options]
 */
export async function update(
  client: MetersEndpointsClientContext,
  meterId: string,
  meter: UpdateRequest,
  options?: UpdateOptions,
): Promise<Meter | void> {
  const path = parse("/openmeter/meters/{meterId}").expand({
    meterId: meterId
  });
  const httpRequestOptions = {
    headers: {},body: jsonUpdateRequestToTransportTransform(meter),
  };
  const response = await client.pathUnchecked(path).put(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonMeterToApplicationTransform(response.body)!;
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
 * Delete a meter.
 *
 * @param {MetersEndpointsClientContext} client
 * @param {string} meterId
 * @param {DeleteOptions} [options]
 */
export async function delete_(
  client: MetersEndpointsClientContext,
  meterId: string,
  options?: DeleteOptions,
): Promise<void> {
  const path = parse("/openmeter/meters/{meterId}").expand({
    meterId: meterId
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
