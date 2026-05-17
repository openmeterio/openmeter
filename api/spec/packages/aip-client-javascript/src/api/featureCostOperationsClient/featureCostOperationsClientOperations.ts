import { parse } from "uri-template";
import {
  FeatureCostOperationsClientContext,
} from "./featureCostOperationsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  jsonFeatureCostQueryResultToApplicationTransform,
  jsonMeterQueryRequestToTransportTransform,
} from "../../models/internal/serializers.js";
import type {
  FeatureCostQueryResult,
  MeterQueryRequest,
} from "../../models/models.js";

export interface QueryCostOptions extends OperationOptions {
  request?: MeterQueryRequest
}
/**
 * Query the cost of a feature.
 *
 * @param {FeatureCostOperationsClientContext} client
 * @param {string} featureId
 * @param {QueryCostOptions} [options]
 */
export async function queryCost(
  client: FeatureCostOperationsClientContext,
  featureId: string,
  options?: QueryCostOptions,
): Promise<FeatureCostQueryResult | void> {
  const path = parse("/query/{featureId}").expand({
    featureId: featureId
  });
  const httpRequestOptions = {
    headers: {

    },body: jsonMeterQueryRequestToTransportTransform(options?.request),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonFeatureCostQueryResultToApplicationTransform(response.body)!;
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
