import { parse } from "uri-template";
import {
  CustomerCreditGrantEndpointsClientContext,
} from "./customerCreditGrantEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  jsonCreditGrantToApplicationTransform,
  jsonUpdateCreditGrantExternalSettlementRequestToTransportTransform,
} from "../../models/internal/serializers.js";
import {
  type CreditGrant,
  UpdateCreditGrantExternalSettlementRequest,
} from "../../models/models.js";

export interface UpdateExternalSettlementOptions extends OperationOptions {}
/**
 * Update the payment settlement status of an externally funded credit grant.
 * Use this endpoint to synchronize the payment state of an external payment
 * with the system so that revenue recognition and credit availability work as
 * expected.
 *
 * @param {CustomerCreditGrantEndpointsClientContext} client
 * @param {string} customerId
 * @param {string} creditGrantId
 * @param {UpdateCreditGrantExternalSettlementRequest} body
 * @param {UpdateExternalSettlementOptions} [options]
 */
export async function updateExternalSettlement(
  client: CustomerCreditGrantEndpointsClientContext,
  customerId: string,
  creditGrantId: string,
  body: UpdateCreditGrantExternalSettlementRequest,
  options?: UpdateExternalSettlementOptions,
): Promise<CreditGrant | void> {
  const path = parse("/openmeter/customers/{customerId}/credits/grants/{creditGrantId}/settlement/external").expand({
    customerId: customerId,
    creditGrantId: creditGrantId
  });
  const httpRequestOptions = {
    headers: {

    },body: jsonUpdateCreditGrantExternalSettlementRequestToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonCreditGrantToApplicationTransform(response.body)!;
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
