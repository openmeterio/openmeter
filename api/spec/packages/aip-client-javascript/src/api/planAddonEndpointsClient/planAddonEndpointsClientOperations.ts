import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import {
  PlanAddonEndpointsClientContext,
} from "./planAddonEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayPlanAddonToApplicationTransform,
  jsonCreateRequestToTransportTransform_11,
  jsonPageMetaToApplicationTransform,
  jsonPlanAddonToApplicationTransform,
  jsonUpsertRequestToTransportTransform_8,
} from "../../models/internal/serializers.js";
import {
  CreateRequest_11 as CreateRequest,
  PlanAddon,
  UpsertRequest_8 as UpsertRequest,
} from "../../models/models.js";

export interface ListPlanAddonsOptions extends OperationOptions {
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
export interface ListPlanAddonsPageSettings {}
export interface ListPlanAddonsPageResponse {
  data: Array<PlanAddon>
}
async function listPlanAddonsSend(
  client: PlanAddonEndpointsClientContext,
  planId: string,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/plans/{planId}/addons{?page*}").expand({
    planId: planId,
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }})
  });
  const httpRequestOptions = {
    headers: {},
  };
  return await client.pathUnchecked(path).get(httpRequestOptions);;
}
function listPlanAddonsDeserialize(
  response: PathUncheckedResponse,
  options?: ListPlanAddonsOptions,
) {
  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return {
      data: jsonArrayPlanAddonToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
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
  if (+response.status === 404 && !response.body) {
    return;
  }
  throw createRestError(response);
}
export function listPlanAddons(
  client: PlanAddonEndpointsClientContext,
  planId: string,
  options?: ListPlanAddonsOptions,
): PagedAsyncIterableIterator<PlanAddon,ListPlanAddonsPageResponse,ListPlanAddonsPageSettings> {
  function getElements(response: ListPlanAddonsPageResponse) {
    return response.data;
  }
  async function getPagedResponse(
    nextToken?: string,
    settings?: ListPlanAddonsPageSettings,
  ) {

            let response: PathUncheckedResponse;
            if (nextToken) {
              response = await client.pathUnchecked(nextToken).get();
            } else {
              const combinedOptions = { ...options, ...settings };
              response = await listPlanAddonsSend(client, planId, combinedOptions);
            }
    return {
    pagedResponse: await listPlanAddonsDeserialize(response, options),
    nextToken: undefined,
    };
  }
  return buildPagedAsyncIterator<PlanAddon, ListPlanAddonsPageResponse, ListPlanAddonsPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface CreatePlanAddonOptions extends OperationOptions {}
/**
 * Add an add-on to a plan.
 *
 * @param {PlanAddonEndpointsClientContext} client
 * @param {string} planId
 * @param {CreateRequest} planAddon
 * @param {CreatePlanAddonOptions} [options]
 */
export async function createPlanAddon(
  client: PlanAddonEndpointsClientContext,
  planId: string,
  planAddon: CreateRequest,
  options?: CreatePlanAddonOptions,
): Promise<PlanAddon | void> {
  const path = parse("/openmeter/plans/{planId}/addons").expand({
    planId: planId
  });
  const httpRequestOptions = {
    headers: {},body: jsonCreateRequestToTransportTransform_11(planAddon),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPlanAddonToApplicationTransform(response.body)!;
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
export interface GetPlanAddonOptions extends OperationOptions {}
/**
 * Get an add-on association for a plan.
 *
 * @param {PlanAddonEndpointsClientContext} client
 * @param {string} planId
 * @param {string} planAddonId
 * @param {GetPlanAddonOptions} [options]
 */
export async function getPlanAddon(
  client: PlanAddonEndpointsClientContext,
  planId: string,
  planAddonId: string,
  options?: GetPlanAddonOptions,
): Promise<PlanAddon | void> {
  const path = parse("/openmeter/plans/{planId}/addons/{planAddonId}").expand({
    planId: planId,
    planAddonId: planAddonId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPlanAddonToApplicationTransform(response.body)!;
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
export interface UpdatePlanAddonOptions extends OperationOptions {}
/**
 * Update an add-on association for a plan.
 *
 * @param {PlanAddonEndpointsClientContext} client
 * @param {string} planId
 * @param {string} planAddonId
 * @param {UpsertRequest} planAddon
 * @param {UpdatePlanAddonOptions} [options]
 */
export async function updatePlanAddon(
  client: PlanAddonEndpointsClientContext,
  planId: string,
  planAddonId: string,
  planAddon: UpsertRequest,
  options?: UpdatePlanAddonOptions,
): Promise<PlanAddon | void> {
  const path = parse("/openmeter/plans/{planId}/addons/{planAddonId}").expand({
    planId: planId,
    planAddonId: planAddonId
  });
  const httpRequestOptions = {
    headers: {},body: jsonUpsertRequestToTransportTransform_8(planAddon),
  };
  const response = await client.pathUnchecked(path).put(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPlanAddonToApplicationTransform(response.body)!;
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
export interface DeletePlanAddonOptions extends OperationOptions {}
/**
 * Remove an add-on from a plan.
 *
 * @param {PlanAddonEndpointsClientContext} client
 * @param {string} planId
 * @param {string} planAddonId
 * @param {DeletePlanAddonOptions} [options]
 */
export async function deletePlanAddon(
  client: PlanAddonEndpointsClientContext,
  planId: string,
  planAddonId: string,
  options?: DeletePlanAddonOptions,
): Promise<void> {
  const path = parse("/openmeter/plans/{planId}/addons/{planAddonId}").expand({
    planId: planId,
    planAddonId: planAddonId
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
