import { parse } from "uri-template";
import type {
  OrganizationDefaultTaxCodesEndpointsClientContext,
} from "./organizationDefaultTaxCodesEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  jsonOrganizationDefaultTaxCodesToApplicationTransform,
  jsonUpdateRequestToTransportTransform_2,
} from "../../models/internal/serializers.js";
import type {
  OrganizationDefaultTaxCodes,
  UpdateRequest_2 as UpdateRequest,
} from "../../models/models.js";

export interface GetOptions extends OperationOptions {}
export async function get(
  client: OrganizationDefaultTaxCodesEndpointsClientContext,
  options?: GetOptions,
): Promise<OrganizationDefaultTaxCodes | void> {
  const path = parse("/openmeter/defaults/tax-codes").expand({});
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonOrganizationDefaultTaxCodesToApplicationTransform(response.body)!;
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
export interface UpdateOptions extends OperationOptions {}
export async function update(
  client: OrganizationDefaultTaxCodesEndpointsClientContext,
  body: UpdateRequest,
  options?: UpdateOptions,
): Promise<OrganizationDefaultTaxCodes | void> {
  const path = parse("/openmeter/defaults/tax-codes").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonUpdateRequestToTransportTransform_2(body),
  };
  const response = await client.pathUnchecked(path).put(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonOrganizationDefaultTaxCodesToApplicationTransform(response.body)!;
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
