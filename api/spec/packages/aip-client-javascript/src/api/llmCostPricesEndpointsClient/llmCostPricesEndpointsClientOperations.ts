import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import {
  LlmCostPricesEndpointsClientContext,
} from "./llmCostPricesEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayPriceToApplicationTransform,
  jsonListPricesParamsFilterToTransportTransform,
  jsonPageMetaToApplicationTransform,
  jsonPriceToApplicationTransform_2 as jsonPriceToApplicationTransform,
  jsonSortQueryToTransportTransform,
} from "../../models/internal/serializers.js";
import {
  type ListPricesParamsFilter,
  Price_2 as Price,
  type SortQuery,
  type StringFieldFilter,
} from "../../models/models.js";

export interface ListPricesOptions extends OperationOptions {
  provider?: StringFieldFilter
  modelId?: StringFieldFilter
  modelName?: StringFieldFilter
  currency?: StringFieldFilter
  source?: StringFieldFilter
  filter?: ListPricesParamsFilter
  sort?: SortQuery
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
export interface ListPricesPageSettings {}
export interface ListPricesPageResponse {
  data: Array<Price>
}
async function listPricesSend(
  client: LlmCostPricesEndpointsClientContext,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/llm-cost/prices{?filter*,sort,page*}").expand({
    ...(options?.filter && {filter: jsonListPricesParamsFilterToTransportTransform(options.filter)}),
    ...(options?.sort && {sort: jsonSortQueryToTransportTransform(options.sort)}),
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }})
  });
  const httpRequestOptions = {
    headers: {},
  };
  return await client.pathUnchecked(path).get(httpRequestOptions);;
}
function listPricesDeserialize(
  response: PathUncheckedResponse,
  options?: ListPricesOptions,
) {
  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return {
      data: jsonArrayPriceToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
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
export function listPrices(
  client: LlmCostPricesEndpointsClientContext,
  options?: ListPricesOptions,
): PagedAsyncIterableIterator<Price,ListPricesPageResponse,ListPricesPageSettings> {
  function getElements(response: ListPricesPageResponse) {
    return response.data;
  }
  async function getPagedResponse(
    nextToken?: string,
    settings?: ListPricesPageSettings,
  ) {

            let response: PathUncheckedResponse;
            if (nextToken) {
              response = await client.pathUnchecked(nextToken).get();
            } else {
              const combinedOptions = { ...options, ...settings };
              response = await listPricesSend(client, combinedOptions);
            }
    return {
    pagedResponse: await listPricesDeserialize(response, options),
    nextToken: undefined,
    };
  }
  return buildPagedAsyncIterator<Price, ListPricesPageResponse, ListPricesPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface GetPriceOptions extends OperationOptions {}
/**
 * Get a specific LLM cost price by ID. Returns the price with overrides applied
 * if any.
 *
 * @param {LlmCostPricesEndpointsClientContext} client
 * @param {string} priceId
 * @param {GetPriceOptions} [options]
 */
export async function getPrice(
  client: LlmCostPricesEndpointsClientContext,
  priceId: string,
  options?: GetPriceOptions,
): Promise<Price | void> {
  const path = parse("/openmeter/llm-cost/prices/{priceId}").expand({
    priceId: priceId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPriceToApplicationTransform(response.body)!;
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
