import { parse } from "uri-template";
import {
  GovernanceOperationsClientContext,
} from "./governanceOperationsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  jsonCursorPaginationQueryPageToTransportTransform,
  jsonGovernanceQueryRequestToTransportTransform,
  jsonGovernanceQueryResponseToApplicationTransform,
} from "../../models/internal/serializers.js";
import {
  type CursorPaginationQueryPage,
  GovernanceQueryRequest,
  type GovernanceQueryResponse,
} from "../../models/models.js";

export interface QueryOptions extends OperationOptions {
  size?: number
  after?: string
  before?: string
  page?: CursorPaginationQueryPage
}
/**
 * Query feature access for a list of customers. The endpoint resolves each
 * provided identifier to a customer and returns the access status for the
 * requested features, plus optional credit balance availability. _Designed to
 * be called on a fixed refresh interval and the query response is intended to
 * be cached._
 *
 * @param {GovernanceOperationsClientContext} client
 * @param {GovernanceQueryRequest} _
 * @param {QueryOptions} [options]
 */
export async function query(
  client: GovernanceOperationsClientContext,
  _: GovernanceQueryRequest,
  options?: QueryOptions,
): Promise<GovernanceQueryResponse | void> {
  const path = parse("/query{?page*}").expand({
    ...(options?.page && {page: jsonCursorPaginationQueryPageToTransportTransform(options.page)})
  });
  const httpRequestOptions = {
    headers: {},body: jsonGovernanceQueryRequestToTransportTransform(_),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonGovernanceQueryResponseToApplicationTransform(response.body)!;
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
