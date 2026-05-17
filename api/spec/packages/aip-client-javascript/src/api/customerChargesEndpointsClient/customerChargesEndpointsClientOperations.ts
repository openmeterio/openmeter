import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import type {
  CustomerChargesEndpointsClientContext,
} from "./customerChargesEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayChargesExpandToTransportTransform,
  jsonArrayChargeToApplicationTransform,
  jsonListCustomerChargesParamsFilterToTransportTransform,
  jsonPageMetaToApplicationTransform,
  jsonSortQueryToTransportTransform,
} from "../../models/internal/serializers.js";
import {
  Charge,
  type ChargesExpand,
  type ListCustomerChargesParamsFilter,
  type SortQuery,
  type StringFieldFilterExact,
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
  status?: StringFieldFilterExact
  filter?: ListCustomerChargesParamsFilter
  expand?: Array<ChargesExpand>
}
export interface ListPageSettings {}
export interface ListPageResponse {
  data: Array<Charge>
}
async function listSend(
  client: CustomerChargesEndpointsClientContext,
  customerId: string,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/customers/{customerId}/charges{?page*,sort,filter*,expand*}").expand({
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }}),
    customerId: customerId,
    ...(options?.sort && {sort: jsonSortQueryToTransportTransform(options.sort)}),
    ...(options?.filter && {filter: jsonListCustomerChargesParamsFilterToTransportTransform(options.filter)}),
    ...(options?.expand && {expand: jsonArrayChargesExpandToTransportTransform(options.expand)})
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
      data: jsonArrayChargeToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
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
  client: CustomerChargesEndpointsClientContext,
  customerId: string,
  options?: ListOptions,
): PagedAsyncIterableIterator<Charge,ListPageResponse,ListPageSettings> {
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
  return buildPagedAsyncIterator<Charge, ListPageResponse, ListPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
