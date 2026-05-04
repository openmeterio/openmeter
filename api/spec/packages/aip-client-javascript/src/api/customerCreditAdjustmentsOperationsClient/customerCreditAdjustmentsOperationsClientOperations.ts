import { parse } from "uri-template";
import {
  CustomerCreditAdjustmentsOperationsClientContext,
} from "./customerCreditAdjustmentsOperationsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  jsonCreateRequestToTransportTransform_3,
  jsonCreditAdjustmentToApplicationTransform,
} from "../../models/internal/serializers.js";
import {
  CreateRequest_3 as CreateRequest,
  type CreditAdjustment,
} from "../../models/models.js";

export interface CreateOptions extends OperationOptions {}
/**
 * A credit adjustment can be used to make manual adjustments to a customer's
 * credit balance. Supported use-cases: - Usage correction
 *
 * @param {CustomerCreditAdjustmentsOperationsClientContext} client
 * @param {string} customerId
 * @param {CreateRequest} creditAdjustment
 * @param {CreateOptions} [options]
 */
export async function create(
  client: CustomerCreditAdjustmentsOperationsClientContext,
  customerId: string,
  creditAdjustment: CreateRequest,
  options?: CreateOptions,
): Promise<CreditAdjustment | void> {
  const path = parse("/{customerId}").expand({
    customerId: customerId
  });
  const httpRequestOptions = {
    headers: {},body: jsonCreateRequestToTransportTransform_3(creditAdjustment),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCreditAdjustmentToApplicationTransform(response.body)!;
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
