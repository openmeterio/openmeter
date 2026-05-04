import { parse } from "uri-template";
import {
  CustomerCreditBalancesOperationsClientContext,
} from "./customerCreditBalancesOperationsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  jsonCreditBalancesToApplicationTransform,
  jsonGetCreditBalanceParamsFilterToTransportTransform,
} from "../../models/internal/serializers.js";
import type {
  CreditBalances,
  CurrencyCode_2,
  GetCreditBalanceParamsFilter,
} from "../../models/models.js";

export interface GetOptions extends OperationOptions {
  currency?: CurrencyCode_2
  filter?: GetCreditBalanceParamsFilter
}
/**
 * Get a credit balance.
 *
 * @param {CustomerCreditBalancesOperationsClientContext} client
 * @param {string} customerId
 * @param {GetOptions} [options]
 */
export async function get(
  client: CustomerCreditBalancesOperationsClientContext,
  customerId: string,
  options?: GetOptions,
): Promise<CreditBalances | void> {
  const path = parse("/{customerId}{?filter*}").expand({
    customerId: customerId,
    ...(options?.filter && {filter: jsonGetCreditBalanceParamsFilterToTransportTransform(options.filter)})
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCreditBalancesToApplicationTransform(response.body)!;
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
