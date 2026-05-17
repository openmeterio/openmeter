import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import {
  CustomerCreditGrantsEndpointsClientContext,
} from "./customerCreditGrantsEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayCreditGrantToApplicationTransform,
  jsonCreateRequestNestedToTransportTransform,
  jsonCreditGrantToApplicationTransform,
  jsonListCreditGrantsParamsFilterToTransportTransform,
  jsonPageMetaToApplicationTransform,
} from "../../models/internal/serializers.js";
import {
  CreateRequestNested,
  CreditGrant,
  type CreditGrantStatus,
  type ListCreditGrantsParamsFilter,
} from "../../models/models.js";

export interface CreateOptions extends OperationOptions {}
/**
 * Create a new credit grant. A credit grant represents an allocation of prepaid
 * credits to a customer.
 *
 * @param {CustomerCreditGrantsEndpointsClientContext} client
 * @param {string} customerId
 * @param {CreateRequestNested} creditGrant
 * @param {CreateOptions} [options]
 */
export async function create(
  client: CustomerCreditGrantsEndpointsClientContext,
  customerId: string,
  creditGrant: CreateRequestNested,
  options?: CreateOptions,
): Promise<CreditGrant | void> {
  const path = parse("/openmeter/customers/{customerId}/credits/grants").expand({
    customerId: customerId
  });
  const httpRequestOptions = {
    headers: {},body: jsonCreateRequestNestedToTransportTransform(creditGrant),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCreditGrantToApplicationTransform(response.body)!;
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
export interface GetOptions extends OperationOptions {}
/**
 * Get a credit grant.
 *
 * @param {CustomerCreditGrantsEndpointsClientContext} client
 * @param {string} customerId
 * @param {string} creditGrantId
 * @param {GetOptions} [options]
 */
export async function get(
  client: CustomerCreditGrantsEndpointsClientContext,
  customerId: string,
  creditGrantId: string,
  options?: GetOptions,
): Promise<CreditGrant | void> {
  const path = parse("/openmeter/customers/{customerId}/credits/grants/{creditGrantId}").expand({
    customerId: customerId,
    creditGrantId: creditGrantId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCreditGrantToApplicationTransform(response.body)!;
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
  status?: CreditGrantStatus
  currency?: string
  filter?: ListCreditGrantsParamsFilter
}
export interface ListPageSettings {}
export interface ListPageResponse {
  data: Array<CreditGrant>
}
async function listSend(
  client: CustomerCreditGrantsEndpointsClientContext,
  customerId: string,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/customers/{customerId}/credits/grants{?page*,filter*}").expand({
    customerId: customerId,
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }}),
    ...(options?.filter && {filter: jsonListCreditGrantsParamsFilterToTransportTransform(options.filter)})
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
      data: jsonArrayCreditGrantToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
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
  client: CustomerCreditGrantsEndpointsClientContext,
  customerId: string,
  options?: ListOptions,
): PagedAsyncIterableIterator<CreditGrant,ListPageResponse,ListPageSettings> {
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
              response = await listSend(client, customerId, combinedOptions);
            }
    return {
    pagedResponse: await listDeserialize(response, options),
    nextToken: undefined,
    };
  }
  return buildPagedAsyncIterator<CreditGrant, ListPageResponse, ListPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
