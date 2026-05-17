import { parse } from "uri-template";
import {
  CurrenciesCustomEndpointsClientContext,
} from "./currenciesCustomEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  jsonCreateRequestToTransportTransform_6,
  jsonCurrencyCustomToApplicationTransform,
} from "../../models/internal/serializers.js";
import {
  CreateRequest_6 as CreateRequest,
  type CurrencyCustom,
} from "../../models/models.js";

export interface CreateOptions extends OperationOptions {}
/**
 * Create a custom currency. This operation allows defining your own custom
 * currency for billing purposes.
 *
 * @param {CurrenciesCustomEndpointsClientContext} client
 * @param {CreateRequest} body
 * @param {CreateOptions} [options]
 */
export async function create(
  client: CurrenciesCustomEndpointsClientContext,
  body: CreateRequest,
  options?: CreateOptions,
): Promise<CurrencyCustom | void> {
  const path = parse("/openmeter/currencies/custom").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonCreateRequestToTransportTransform_6(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCurrencyCustomToApplicationTransform(response.body)!;
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
