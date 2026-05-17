import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import type {
  CustomersEndpointsClientContext,
} from "./customersEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayCustomerToApplicationTransform,
  jsonCreateRequestToTransportTransform_2,
  jsonCustomerToApplicationTransform,
  jsonListCustomersParamsFilterToTransportTransform,
  jsonPageMetaToApplicationTransform,
  jsonSortQueryToTransportTransform,
  jsonUpsertRequestToTransportTransform,
} from "../../models/internal/serializers.js";
import {
  type CreateRequest_2 as CreateRequest,
  Customer,
  type ListCustomersParamsFilter,
  type SortQuery,
  type StringFieldFilter,
  type UlidFieldFilter,
  type UpsertRequest,
} from "../../models/models.js";

export interface CreateOptions extends OperationOptions {}
export async function create(
  client: CustomersEndpointsClientContext,
  customer: CreateRequest,
  options?: CreateOptions,
): Promise<Customer | void> {
  const path = parse("/openmeter/customers").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonCreateRequestToTransportTransform_2(customer),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCustomerToApplicationTransform(response.body)!;
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
export async function get(
  client: CustomersEndpointsClientContext,
  customerId: string,
  options?: GetOptions,
): Promise<Customer | void> {
  const path = parse("/openmeter/customers/{customerId}").expand({
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
    return jsonCustomerToApplicationTransform(response.body)!;
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
  primaryEmail?: StringFieldFilter
  usageAttributionSubjectKey?: StringFieldFilter
  planKey?: StringFieldFilter
  billingProfileId?: UlidFieldFilter
  filter?: ListCustomersParamsFilter
}
export interface ListPageSettings {}
export interface ListPageResponse {
  data: Array<Customer>
}
async function listSend(
  client: CustomersEndpointsClientContext,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/customers{?page*,sort,filter*}").expand({
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }}),
    ...(options?.sort && {sort: jsonSortQueryToTransportTransform(options.sort)}),
    ...(options?.filter && {filter: jsonListCustomersParamsFilterToTransportTransform(options.filter)})
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
      data: jsonArrayCustomerToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
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
  client: CustomersEndpointsClientContext,
  options?: ListOptions,
): PagedAsyncIterableIterator<Customer,ListPageResponse,ListPageSettings> {
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
  return buildPagedAsyncIterator<Customer, ListPageResponse, ListPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface UpsertOptions extends OperationOptions {}
export async function upsert(
  client: CustomersEndpointsClientContext,
  customerId: string,
  customer: UpsertRequest,
  options?: UpsertOptions,
): Promise<Customer | void> {
  const path = parse("/openmeter/customers/{customerId}").expand({
    customerId: customerId
  });
  const httpRequestOptions = {
    headers: {},body: jsonUpsertRequestToTransportTransform(customer),
  };
  const response = await client.pathUnchecked(path).put(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCustomerToApplicationTransform(response.body)!;
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
export interface DeleteOptions extends OperationOptions {}
export async function delete_(
  client: CustomersEndpointsClientContext,
  customerId: string,
  options?: DeleteOptions,
): Promise<void> {
  const path = parse("/openmeter/customers/{customerId}").expand({
    customerId: customerId
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
