import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import {
  AddonsEndpointsClientContext,
} from "./addonsEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonAddonToApplicationTransform,
  jsonArrayAddonToApplicationTransform,
  jsonCreateRequestToTransportTransform_10,
  jsonListAddonsParamsFilterToTransportTransform,
  jsonPageMetaToApplicationTransform,
  jsonSortQueryToTransportTransform,
  jsonUpsertRequestToTransportTransform_7,
} from "../../models/internal/serializers.js";
import {
  Addon,
  CreateRequest_10 as CreateRequest,
  type ListAddonsParamsFilter,
  type SortQuery,
  type StringFieldFilter,
  type StringFieldFilterExact,
  type UlidFieldFilter,
  UpsertRequest_7 as UpsertRequest,
} from "../../models/models.js";

export interface ListAddonsOptions extends OperationOptions {
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
  key?: StringFieldFilter
  name?: StringFieldFilter
  status?: StringFieldFilterExact
  currency?: StringFieldFilterExact
  filter?: ListAddonsParamsFilter
}
export interface ListAddonsPageSettings {}
export interface ListAddonsPageResponse {
  data: Array<Addon>
}
async function listAddonsSend(
  client: AddonsEndpointsClientContext,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/addons{?page*,sort,filter*}").expand({
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }}),
    ...(options?.sort && {sort: jsonSortQueryToTransportTransform(options.sort)}),
    ...(options?.filter && {filter: jsonListAddonsParamsFilterToTransportTransform(options.filter)})
  });
  const httpRequestOptions = {
    headers: {},
  };
  return await client.pathUnchecked(path).get(httpRequestOptions);;
}
function listAddonsDeserialize(
  response: PathUncheckedResponse,
  options?: ListAddonsOptions,
) {
  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return {
      data: jsonArrayAddonToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
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
export function listAddons(
  client: AddonsEndpointsClientContext,
  options?: ListAddonsOptions,
): PagedAsyncIterableIterator<Addon,ListAddonsPageResponse,ListAddonsPageSettings> {
  function getElements(response: ListAddonsPageResponse) {
    return response.data;
  }
  async function getPagedResponse(
    nextToken?: string,
    settings?: ListAddonsPageSettings,
  ) {

            let response: PathUncheckedResponse;
            if (nextToken) {
              response = await client.pathUnchecked(nextToken).get();
            } else {
              const combinedOptions = { ...options, ...settings };
              response = await listAddonsSend(client, combinedOptions);
            }
    return {
    pagedResponse: await listAddonsDeserialize(response, options),
    nextToken: undefined,
    };
  }
  return buildPagedAsyncIterator<Addon, ListAddonsPageResponse, ListAddonsPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface CreateAddonOptions extends OperationOptions {}
/**
 * Create a new add-on.
 *
 * @param {AddonsEndpointsClientContext} client
 * @param {CreateRequest} addon
 * @param {CreateAddonOptions} [options]
 */
export async function createAddon(
  client: AddonsEndpointsClientContext,
  addon: CreateRequest,
  options?: CreateAddonOptions,
): Promise<Addon | void> {
  const path = parse("/openmeter/addons").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonCreateRequestToTransportTransform_10(addon),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonAddonToApplicationTransform(response.body)!;
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
export interface UpdateAddonOptions extends OperationOptions {}
/**
 * Update an add-on by id.
 *
 * @param {AddonsEndpointsClientContext} client
 * @param {string} addonId
 * @param {UpsertRequest} addon
 * @param {UpdateAddonOptions} [options]
 */
export async function updateAddon(
  client: AddonsEndpointsClientContext,
  addonId: string,
  addon: UpsertRequest,
  options?: UpdateAddonOptions,
): Promise<Addon | void> {
  const path = parse("/openmeter/addons/{addonId}").expand({
    addonId: addonId
  });
  const httpRequestOptions = {
    headers: {},body: jsonUpsertRequestToTransportTransform_7(addon),
  };
  const response = await client.pathUnchecked(path).put(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonAddonToApplicationTransform(response.body)!;
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
export interface GetAddonOptions extends OperationOptions {}
/**
 * Get add-on by id.
 *
 * @param {AddonsEndpointsClientContext} client
 * @param {string} addonId
 * @param {GetAddonOptions} [options]
 */
export async function getAddon(
  client: AddonsEndpointsClientContext,
  addonId: string,
  options?: GetAddonOptions,
): Promise<Addon | void> {
  const path = parse("/openmeter/addons/{addonId}").expand({
    addonId: addonId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonAddonToApplicationTransform(response.body)!;
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
export interface DeleteAddonOptions extends OperationOptions {}
/**
 * Soft delete add-on by id.
 *
 * @param {AddonsEndpointsClientContext} client
 * @param {string} addonId
 * @param {DeleteAddonOptions} [options]
 */
export async function deleteAddon(
  client: AddonsEndpointsClientContext,
  addonId: string,
  options?: DeleteAddonOptions,
): Promise<void> {
  const path = parse("/openmeter/addons/{addonId}").expand({
    addonId: addonId
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
export interface ArchiveAddonOptions extends OperationOptions {}
/**
 * Archive an add-on version.
 *
 * @param {AddonsEndpointsClientContext} client
 * @param {string} addonId
 * @param {ArchiveAddonOptions} [options]
 */
export async function archiveAddon(
  client: AddonsEndpointsClientContext,
  addonId: string,
  options?: ArchiveAddonOptions,
): Promise<Addon | void> {
  const path = parse("/openmeter/addons/{addonId}/archive").expand({
    addonId: addonId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonAddonToApplicationTransform(response.body)!;
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
export interface PublishAddonOptions extends OperationOptions {}
/**
 * Publish an add-on version.
 *
 * @param {AddonsEndpointsClientContext} client
 * @param {string} addonId
 * @param {PublishAddonOptions} [options]
 */
export async function publishAddon(
  client: AddonsEndpointsClientContext,
  addonId: string,
  options?: PublishAddonOptions,
): Promise<Addon | void> {
  const path = parse("/openmeter/addons/{addonId}/publish").expand({
    addonId: addonId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonAddonToApplicationTransform(response.body)!;
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
