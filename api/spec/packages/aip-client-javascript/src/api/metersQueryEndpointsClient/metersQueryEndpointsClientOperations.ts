import { parse } from "uri-template";
import {
  MetersQueryEndpointsClientContext,
} from "./metersQueryEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  jsonMeterQueryRequestToTransportTransform,
  jsonMeterQueryResultToApplicationTransform,
} from "../../models/internal/serializers.js";
import {
  MeterQueryRequest,
  type MeterQueryResult,
} from "../../models/models.js";

export interface QueryOptions extends OperationOptions {}
/**
 * Query a meter for usage. Set `Accept: application/json` (the default) to get
 * a structured JSON response. Set `Accept: text/csv` to download the same data
 * as a CSV file suitable for spreadsheets. The CSV columns, in order, are:
 * `from, to, [subject,] [customer_id, customer_key, customer_name,]
 * <dimensions...>, value` The `subject` column is emitted only when `subject`
 * is in the query's `group_by_dimensions`. The three `customer_*` columns are
 * emitted together only when `customer_id` is in the query's
 * `group_by_dimensions`.
 *
 * @param {MetersQueryEndpointsClientContext} client
 * @param {string} meterId
 * @param {MeterQueryRequest} request
 * @param {QueryOptions} [options]
 */
export async function query(
  client: MetersQueryEndpointsClientContext,
  meterId: string,
  request: MeterQueryRequest,
  options?: QueryOptions,
): Promise<MeterQueryResult | void> {
  const path = parse("/openmeter/meters/{meterId}/query").expand({
    meterId: meterId
  });
  const httpRequestOptions = {
    headers: {},body: jsonMeterQueryRequestToTransportTransform(request),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonMeterQueryResultToApplicationTransform(response.body)!;
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
export interface QueryCsvOptions extends OperationOptions {}
export async function queryCsv(
  client: MetersQueryEndpointsClientContext,
  meterId: string,
  options?: QueryCsvOptions,
): Promise<string | void> {
  const path = parse("/openmeter/meters/{meterId}/query").expand({
    meterId: meterId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("text/csv")) {
    return response.body!;
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
