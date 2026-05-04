import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import {
  FeaturesEndpointsClientContext,
} from "./featuresEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayFeatureToApplicationTransform,
  jsonCreateRequestToTransportTransform_8,
  jsonFeatureToApplicationTransform,
  jsonFeatureUpdateRequestToTransportTransform,
  jsonListFeaturesParamsFilterToTransportTransform,
  jsonPageMetaToApplicationTransform,
  jsonSortQueryToTransportTransform,
} from "../../models/internal/serializers.js";
import {
  CreateRequest_8 as CreateRequest,
  Feature,
  FeatureUpdateRequest,
  type ListFeaturesParamsFilter,
  type SortQuery,
  type StringFieldFilter,
  type UlidFieldFilter,
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
  sort?: SortQuery
  meterId?: UlidFieldFilter
  key?: StringFieldFilter
  name?: StringFieldFilter
  filter?: ListFeaturesParamsFilter
}
export interface ListPageSettings {}
export interface ListPageResponse {
  data: Array<Feature>
}
async function listSend(
  client: FeaturesEndpointsClientContext,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/features{?page*,sort,filter*}").expand({
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }}),
    ...(options?.sort && {sort: jsonSortQueryToTransportTransform(options.sort)}),
    ...(options?.filter && {filter: jsonListFeaturesParamsFilterToTransportTransform(options.filter)})
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
      data: jsonArrayFeatureToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
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
  client: FeaturesEndpointsClientContext,
  options?: ListOptions,
): PagedAsyncIterableIterator<Feature,ListPageResponse,ListPageSettings> {
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
  return buildPagedAsyncIterator<Feature, ListPageResponse, ListPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface CreateOptions extends OperationOptions {}
/**
 * Create a feature.
 *
 * @param {FeaturesEndpointsClientContext} client
 * @param {CreateRequest} feature
 * @param {CreateOptions} [options]
 */
export async function create(
  client: FeaturesEndpointsClientContext,
  feature: CreateRequest,
  options?: CreateOptions,
): Promise<Feature | void> {
  const path = parse("/openmeter/features").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonCreateRequestToTransportTransform_8(feature),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonFeatureToApplicationTransform(response.body)!;
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
 * Get a feature by id.
 *
 * @param {FeaturesEndpointsClientContext} client
 * @param {string} featureId
 * @param {GetOptions} [options]
 */
export async function get(
  client: FeaturesEndpointsClientContext,
  featureId: string,
  options?: GetOptions,
): Promise<Feature | void> {
  const path = parse("/openmeter/features/{featureId}").expand({
    featureId: featureId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonFeatureToApplicationTransform(response.body)!;
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
  if (+response.status === 410 && !response.body) {
    return;
  }
  if (+response.status === 404 && !response.body) {
    return;
  }
  throw createRestError(response);
}
;
export interface UpdateOptions extends OperationOptions {}
/**
 * Update a feature by id. Currently only the unit_cost field can be updated.
 *
 * @param {FeaturesEndpointsClientContext} client
 * @param {string} featureId
 * @param {FeatureUpdateRequest} feature
 * @param {UpdateOptions} [options]
 */
export async function update(
  client: FeaturesEndpointsClientContext,
  featureId: string,
  feature: FeatureUpdateRequest,
  options?: UpdateOptions,
): Promise<Feature | void> {
  const path = parse("/openmeter/features/{featureId}").expand({
    featureId: featureId
  });
  const httpRequestOptions = {
    headers: {},body: jsonFeatureUpdateRequestToTransportTransform(feature),
  };
  const response = await client.pathUnchecked(path).patch(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonFeatureToApplicationTransform(response.body)!;
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
  if (+response.status === 404 && !response.body) {
    return;
  }
  throw createRestError(response);
}
;
export interface DeleteOptions extends OperationOptions {}
/**
 * Delete a feature by id.
 *
 * @param {FeaturesEndpointsClientContext} client
 * @param {string} featureId
 * @param {DeleteOptions} [options]
 */
export async function delete_(
  client: FeaturesEndpointsClientContext,
  featureId: string,
  options?: DeleteOptions,
): Promise<void> {
  const path = parse("/openmeter/features/{featureId}").expand({
    featureId: featureId
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
  if (+response.status === 400 && !response.body) {
    return;
  }
  if (+response.status === 401 && !response.body) {
    return;
  }
  if (+response.status === 403 && !response.body) {
    return;
  }
  if (+response.status === 404 && !response.body) {
    return;
  }
  throw createRestError(response);
}
;
