import { parse } from "uri-template";
import type {
  CustomerEntitlementsOperationsClientContext,
} from "./customerEntitlementsOperationsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  jsonListCustomerEntitlementAccessResponseDataToApplicationTransform,
} from "../../models/internal/serializers.js";
import type {
  ListCustomerEntitlementAccessResponseData,
} from "../../models/models.js";

export interface ListOptions extends OperationOptions {}
export async function list(
  client: CustomerEntitlementsOperationsClientContext,
  customerId: string,
  options?: ListOptions,
): Promise<ListCustomerEntitlementAccessResponseData | void> {
  const path = parse("/{customerId}").expand({
    customerId: customerId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonListCustomerEntitlementAccessResponseDataToApplicationTransform(response.body)!;
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
