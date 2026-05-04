import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import type {
  CustomerCreditTransactionOperationsClientContext,
} from "./customerCreditTransactionOperationsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayCreditTransactionToApplicationTransform,
  jsonCursorMetaToApplicationTransform,
  jsonCursorPaginationQueryPageToTransportTransform,
  jsonListCreditTransactionsParamsFilterToTransportTransform,
} from "../../models/internal/serializers.js";
import {
  CreditTransaction,
  type CreditTransactionType,
  type CurrencyCode_2,
  type CursorPaginationQueryPage,
  type ListCreditTransactionsParamsFilter,
} from "../../models/models.js";

export interface ListOptions extends OperationOptions {
  size?: number
  after?: string
  before?: string
  page?: CursorPaginationQueryPage
  type?: CreditTransactionType
  currency?: CurrencyCode_2
  filter?: ListCreditTransactionsParamsFilter
}
export interface ListPageSettings {}
export interface ListPageResponse {
  data: Array<CreditTransaction>
}
async function listSend(
  client: CustomerCreditTransactionOperationsClientContext,
  customerId: string,
  options?: Record<string, any>,
) {
  const path = parse("/{customerId}{?page*,filter*}").expand({
    customerId: customerId,
    ...(options?.page && {page: jsonCursorPaginationQueryPageToTransportTransform(options.page)}),
    ...(options?.filter && {filter: jsonListCreditTransactionsParamsFilterToTransportTransform(options.filter)})
  });
  const httpRequestOptions = {
    headers: {},
  };
  return await client.pathUnchecked(path).get(httpRequestOptions);;
}
function listDeserialize(
  response: PathUncheckedResponse,
  options?: ListOptions,
) {
  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return {
      data: jsonArrayCreditTransactionToApplicationTransform(response.body.data),meta: jsonCursorMetaToApplicationTransform(response.body.meta)
    }!;
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
export function list(
  client: CustomerCreditTransactionOperationsClientContext,
  customerId: string,
  options?: ListOptions,
): PagedAsyncIterableIterator<CreditTransaction,ListPageResponse,ListPageSettings> {
  function getElements(response: ListPageResponse) {
    return response.data;
  }
  async function getPagedResponse(
    nextToken?: string,
    settings?: ListPageSettings,
  ) {

            let response: PathUncheckedResponse;
            if (nextToken) {
              response = await client.pathUnchecked(nextToken).get();
            } else {
              const combinedOptions = { ...options, ...settings };
              response = await listSend(client, customerId, combinedOptions);
            }
    return {
    pagedResponse: await listDeserialize(response, options),
    nextToken: undefined,
    };
  }
  return buildPagedAsyncIterator<CreditTransaction, ListPageResponse, ListPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
