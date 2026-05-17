import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import { PlansEndpointsClientContext } from "./plansEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayPlanToApplicationTransform,
  jsonCreateRequestToTransportTransform_9,
  jsonListPlansParamsFilterToTransportTransform,
  jsonPageMetaToApplicationTransform,
  jsonPlanToApplicationTransform,
  jsonSortQueryToTransportTransform,
  jsonUpsertRequestToTransportTransform_6,
} from "../../models/internal/serializers.js";
import {
  CreateRequest_9 as CreateRequest,
  type ListPlansParamsFilter,
  Plan,
  type SortQuery,
  type StringFieldFilter,
  type StringFieldFilterExact,
  UpsertRequest_6 as UpsertRequest,
} from "../../models/models.js";

export interface ListPlansOptions extends OperationOptions {
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
  status?: StringFieldFilterExact
  currency?: StringFieldFilterExact
  filter?: ListPlansParamsFilter
}
export interface ListPlansPageSettings {}
export interface ListPlansPageResponse {
  data: Array<Plan>
}
async function listPlansSend(
  client: PlansEndpointsClientContext,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/plans{?page*,sort,filter*}").expand({
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }}),
    ...(options?.sort && {sort: jsonSortQueryToTransportTransform(options.sort)}),
    ...(options?.filter && {filter: jsonListPlansParamsFilterToTransportTransform(options.filter)})
  });
  const httpRequestOptions = {
    headers: {},
  };
  return await client.pathUnchecked(path).get(httpRequestOptions);;
}
function listPlansDeserialize(
  response: PathUncheckedResponse,
  options?: ListPlansOptions,
) {
  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return {
      data: jsonArrayPlanToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
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
export function listPlans(
  client: PlansEndpointsClientContext,
  options?: ListPlansOptions,
): PagedAsyncIterableIterator<Plan,ListPlansPageResponse,ListPlansPageSettings> {
  function getElements(response: ListPlansPageResponse) {
    return response.data;
  }
  async function getPagedResponse(
    nextToken?: string,
    settings?: ListPlansPageSettings,
  ) {

            let response: PathUncheckedResponse;
            if (nextToken) {
              response = await client.pathUnchecked(nextToken).get();
            } else {
              const combinedOptions = { ...options, ...settings };
              response = await listPlansSend(client, combinedOptions);
            }
    return {
    pagedResponse: await listPlansDeserialize(response, options),
    nextToken: undefined,
    };
  }
  return buildPagedAsyncIterator<Plan, ListPlansPageResponse, ListPlansPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface CreatePlanOptions extends OperationOptions {}
/**
 * Create a new plan.
 *
 * @param {PlansEndpointsClientContext} client
 * @param {CreateRequest} plan
 * @param {CreatePlanOptions} [options]
 */
export async function createPlan(
  client: PlansEndpointsClientContext,
  plan: CreateRequest,
  options?: CreatePlanOptions,
): Promise<Plan | void> {
  const path = parse("/openmeter/plans").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonCreateRequestToTransportTransform_9(plan),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPlanToApplicationTransform(response.body)!;
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
export interface UpdatePlanOptions extends OperationOptions {}
/**
 * Update a plan by id.
 *
 * @param {PlansEndpointsClientContext} client
 * @param {string} planId
 * @param {UpsertRequest} plan
 * @param {UpdatePlanOptions} [options]
 */
export async function updatePlan(
  client: PlansEndpointsClientContext,
  planId: string,
  plan: UpsertRequest,
  options?: UpdatePlanOptions,
): Promise<Plan | void> {
  const path = parse("/openmeter/plans/{planId}").expand({
    planId: planId
  });
  const httpRequestOptions = {
    headers: {},body: jsonUpsertRequestToTransportTransform_6(plan),
  };
  const response = await client.pathUnchecked(path).put(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPlanToApplicationTransform(response.body)!;
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
export interface GetPlanOptions extends OperationOptions {}
/**
 * Get a plan by id.
 *
 * @param {PlansEndpointsClientContext} client
 * @param {string} planId
 * @param {GetPlanOptions} [options]
 */
export async function getPlan(
  client: PlansEndpointsClientContext,
  planId: string,
  options?: GetPlanOptions,
): Promise<Plan | void> {
  const path = parse("/openmeter/plans/{planId}").expand({
    planId: planId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPlanToApplicationTransform(response.body)!;
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
export interface DeletePlanOptions extends OperationOptions {}
/**
 * Delete a plan by id.
 *
 * @param {PlansEndpointsClientContext} client
 * @param {string} planId
 * @param {DeletePlanOptions} [options]
 */
export async function deletePlan(
  client: PlansEndpointsClientContext,
  planId: string,
  options?: DeletePlanOptions,
): Promise<void> {
  const path = parse("/openmeter/plans/{planId}").expand({
    planId: planId
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
export interface ArchivePlanOptions extends OperationOptions {}
/**
 * Archive a plan version.
 *
 * @param {PlansEndpointsClientContext} client
 * @param {string} planId
 * @param {ArchivePlanOptions} [options]
 */
export async function archivePlan(
  client: PlansEndpointsClientContext,
  planId: string,
  options?: ArchivePlanOptions,
): Promise<Plan | void> {
  const path = parse("/openmeter/plans/{planId}/archive").expand({
    planId: planId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPlanToApplicationTransform(response.body)!;
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
export interface PublishPlanOptions extends OperationOptions {}
/**
 * Publish a plan version.
 *
 * @param {PlansEndpointsClientContext} client
 * @param {string} planId
 * @param {PublishPlanOptions} [options]
 */
export async function publishPlan(
  client: PlansEndpointsClientContext,
  planId: string,
  options?: PublishPlanOptions,
): Promise<Plan | void> {
  const path = parse("/openmeter/plans/{planId}/publish").expand({
    planId: planId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPlanToApplicationTransform(response.body)!;
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
