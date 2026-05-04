import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import {
  CurrenciesCustomCostBasesEndpointsClientContext,
} from "./currenciesCustomCostBasesEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayCostBasisToApplicationTransform,
  jsonCostBasisToApplicationTransform,
  jsonCreateRequestToTransportTransform_7,
  jsonListCostBasesParamsFilterToTransportTransform,
  jsonPageMetaToApplicationTransform,
} from "../../models/internal/serializers.js";
import {
  CostBasis,
  CreateRequest_7 as CreateRequest,
  type ListCostBasesParamsFilter,
} from "../../models/models.js";

export interface GetCostBasesOptions extends OperationOptions {
  fiatCode?: string
  filter?: ListCostBasesParamsFilter
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
export interface GetCostBasesPageSettings {}
export interface GetCostBasesPageResponse {
  data: Array<CostBasis>
}
async function getCostBasesSend(
  client: CurrenciesCustomCostBasesEndpointsClientContext,
  currencyId: string,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/currencies/custom/{currencyId}/cost-bases{?filter*,page*}").expand({
    currencyId: currencyId,
    ...(options?.filter && {filter: jsonListCostBasesParamsFilterToTransportTransform(options.filter)}),
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }})
  });
  const httpRequestOptions = {
    headers: {},
  };
  return await client.pathUnchecked(path).get(httpRequestOptions);;
}
function getCostBasesDeserialize(
  response: PathUncheckedResponse,
  options?: GetCostBasesOptions,
) {
  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return {
      data: jsonArrayCostBasisToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
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
export function getCostBases(
  client: CurrenciesCustomCostBasesEndpointsClientContext,
  currencyId: string,
  options?: GetCostBasesOptions,
): PagedAsyncIterableIterator<CostBasis,GetCostBasesPageResponse,GetCostBasesPageSettings> {
  function getElements(response: GetCostBasesPageResponse) {
    return response.data;
  }
  async function getPagedResponse(
    nextToken?: string,
    settings?: GetCostBasesPageSettings,
  ) {

            let response: PathUncheckedResponse;
            if (nextToken) {
              response = await client.pathUnchecked(nextToken).get();
            } else {
              const combinedOptions = { ...options, ...settings };
              response = await getCostBasesSend(client, currencyId, combinedOptions);
            }
    return {
    pagedResponse: await getCostBasesDeserialize(response, options),
    nextToken: undefined,
    };
  }
  return buildPagedAsyncIterator<CostBasis, GetCostBasesPageResponse, GetCostBasesPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface CreateCostBasisOptions extends OperationOptions {}
/**
 * Create a cost basis for a currency.
 *
 * @param {CurrenciesCustomCostBasesEndpointsClientContext} client
 * @param {string} currencyId
 * @param {CreateRequest} body
 * @param {CreateCostBasisOptions} [options]
 */
export async function createCostBasis(
  client: CurrenciesCustomCostBasesEndpointsClientContext,
  currencyId: string,
  body: CreateRequest,
  options?: CreateCostBasisOptions,
): Promise<CostBasis | void> {
  const path = parse("/openmeter/currencies/custom/{currencyId}/cost-bases").expand({
    currencyId: currencyId
  });
  const httpRequestOptions = {
    headers: {},body: jsonCreateRequestToTransportTransform_7(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCostBasisToApplicationTransform(response.body)!;
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
