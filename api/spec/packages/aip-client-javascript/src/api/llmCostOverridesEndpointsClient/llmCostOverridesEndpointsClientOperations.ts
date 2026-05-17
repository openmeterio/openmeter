import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import {
  LlmCostOverridesEndpointsClientContext,
} from "./llmCostOverridesEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayPriceToApplicationTransform,
  jsonListPricesParamsFilterToTransportTransform,
  jsonOverrideCreateToTransportTransform,
  jsonPageMetaToApplicationTransform,
  jsonPriceToApplicationTransform_2 as jsonPriceToApplicationTransform,
} from "../../models/internal/serializers.js";
import {
  type ListPricesParamsFilter,
  OverrideCreate,
  Price_2 as Price,
  type StringFieldFilter,
} from "../../models/models.js";

export interface ListOverridesOptions extends OperationOptions {
  provider?: StringFieldFilter
  modelId?: StringFieldFilter
  modelName?: StringFieldFilter
  currency?: StringFieldFilter
  source?: StringFieldFilter
  filter?: ListPricesParamsFilter
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
export interface ListOverridesPageSettings {}
export interface ListOverridesPageResponse {
  data: Array<Price>
}
async function listOverridesSend(
  client: LlmCostOverridesEndpointsClientContext,
  options?: Record<string, any>,
) {
  const path = parse("/openmeter/llm-cost/overrides{?filter*,page*}").expand({
    ...(options?.filter && {filter: jsonListPricesParamsFilterToTransportTransform(options.filter)}),
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }})
  });
  const httpRequestOptions = {
    headers: {},
  };
  return await client.pathUnchecked(path).get(httpRequestOptions);;
}
function listOverridesDeserialize(
  response: PathUncheckedResponse,
  options?: ListOverridesOptions,
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
export function listOverrides(
  client: LlmCostOverridesEndpointsClientContext,
  options?: ListOverridesOptions,
): PagedAsyncIterableIterator<Price,ListOverridesPageResponse,ListOverridesPageSettings> {
  function getElements(response: ListOverridesPageResponse) {
    return response.data;
  }
  async function getPagedResponse(
    nextToken?: string,
    settings?: ListOverridesPageSettings,
  ) {

            let response: PathUncheckedResponse;
            if (nextToken) {
              response = await client.pathUnchecked(nextToken).get();
            } else {
              const combinedOptions = { ...options, ...settings };
              response = await listOverridesSend(client, combinedOptions);
            }
    return {
    pagedResponse: await listOverridesDeserialize(response, options),
    nextToken: undefined,
    };
  }
  return buildPagedAsyncIterator<Price, ListOverridesPageResponse, ListOverridesPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface CreateOverrideOptions extends OperationOptions {}
/**
 * Create a per-namespace price override.
 *
 * @param {LlmCostOverridesEndpointsClientContext} client
 * @param {OverrideCreate} body
 * @param {CreateOverrideOptions} [options]
 */
export async function createOverride(
  client: LlmCostOverridesEndpointsClientContext,
  body: OverrideCreate,
  options?: CreateOverrideOptions,
): Promise<Price | void> {
  const path = parse("/openmeter/llm-cost/overrides").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonOverrideCreateToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonPriceToApplicationTransform(response.body)!;
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
export interface DeleteOverrideOptions extends OperationOptions {}
/**
 * Delete a per-namespace price override.
 *
 * @param {LlmCostOverridesEndpointsClientContext} client
 * @param {string} priceId
 * @param {DeleteOverrideOptions} [options]
 */
export async function deleteOverride(
  client: LlmCostOverridesEndpointsClientContext,
  priceId: string,
  options?: DeleteOverrideOptions,
): Promise<void> {
  const path = parse("/openmeter/llm-cost/overrides/{priceId}").expand({
    priceId: priceId
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
