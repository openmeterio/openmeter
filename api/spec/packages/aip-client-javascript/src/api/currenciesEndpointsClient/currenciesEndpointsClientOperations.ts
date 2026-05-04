import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import type {
  CurrenciesEndpointsClientContext,
} from "./currenciesEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayCurrencyToApplicationTransform,
  jsonListCurrenciesParamsFilterToTransportTransform,
  jsonPageMetaToApplicationTransform,
} from "../../models/internal/serializers.js";
import {
  Currency,
  type CurrencyType,
  type ListCurrenciesParamsFilter,
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
  type?: CurrencyType
  filter?: ListCurrenciesParamsFilter
}
export interface ListPageSettings {}
export interface ListPageResponse {
  data: Array<Currency>
}
async function listSend(
  client: CurrenciesEndpointsClientContext,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/currencies{?page*,filter*}").expand({
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }}),
    ...(options?.filter && {filter: jsonListCurrenciesParamsFilterToTransportTransform(options.filter)})
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
      data: jsonArrayCurrencyToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
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
  client: CurrenciesEndpointsClientContext,
  options?: ListOptions,
): PagedAsyncIterableIterator<Currency,ListPageResponse,ListPageSettings> {
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
  return buildPagedAsyncIterator<Currency, ListPageResponse, ListPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
