import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import type {
  TaxCodesOperationsClientContext,
} from "./taxCodesOperationsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayTaxCodeToApplicationTransform,
  jsonCreateRequestToTransportTransform_5,
  jsonPageMetaToApplicationTransform,
  jsonTaxCodeToApplicationTransform,
  jsonUpsertRequestToTransportTransform_5,
} from "../../models/internal/serializers.js";
import {
  type CreateRequest_5 as CreateRequest,
  TaxCode,
  type UpsertRequest_5 as UpsertRequest,
} from "../../models/models.js";

export interface CreateOptions extends OperationOptions {}
export async function create(
  client: TaxCodesOperationsClientContext,
  taxCode: CreateRequest,
  options?: CreateOptions,
): Promise<TaxCode | void> {
  const path = parse("/").expand({});
  const httpRequestOptions = {
    headers: {},body: jsonCreateRequestToTransportTransform_5(taxCode),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 201 && response.headers["content-type"]?.includes("application/json")) {
    return jsonTaxCodeToApplicationTransform(response.body)!;
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
export interface GetOptions extends OperationOptions {}
export async function get(
  client: TaxCodesOperationsClientContext,
  taxCodeId: string,
  options?: GetOptions,
): Promise<TaxCode | void> {
  const path = parse("/{taxCodeId}").expand({
    taxCodeId: taxCodeId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonTaxCodeToApplicationTransform(response.body)!;
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
export interface ListOptions extends OperationOptions {
  size?: number
  number?: number
  page?: {
    /**
     * The number of items to include per page.
     */
    size?: number;
    /**
     * The page number.
     */
    number?: number;
  }
  includeDeleted?: boolean
}
export interface ListPageSettings {}
export interface ListPageResponse {
  data: Array<TaxCode>
}
async function listSend(
  client: TaxCodesOperationsClientContext,
  options?: Record<string, any>,
) {
  const path = parse("/{?page*,include_deleted}").expand({
    ...(options?.page && {page: {
      size: options.page.size,number: options.page.number
    }}),
    ...(options?.includeDeleted && {include_deleted: options.includeDeleted})
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
      data: jsonArrayTaxCodeToApplicationTransform(response.body.data),meta: jsonPageMetaToApplicationTransform(response.body.meta)
    }!;
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
  client: TaxCodesOperationsClientContext,
  options?: ListOptions,
): PagedAsyncIterableIterator<TaxCode,ListPageResponse,ListPageSettings> {
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
              response = await listSend(client, combinedOptions);
            }
    return {
    pagedResponse: await listDeserialize(response, options),
    nextToken: undefined,
    };
  }
  return buildPagedAsyncIterator<TaxCode, ListPageResponse, ListPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface UpsertOptions extends OperationOptions {}
export async function upsert(
  client: TaxCodesOperationsClientContext,
  taxCodeId: string,
  taxCode: UpsertRequest,
  options?: UpsertOptions,
): Promise<TaxCode | void> {
  const path = parse("/{taxCodeId}").expand({
    taxCodeId: taxCodeId
  });
  const httpRequestOptions = {
    headers: {},body: jsonUpsertRequestToTransportTransform_5(taxCode),
  };
  const response = await client.pathUnchecked(path).put(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 200 && response.headers["content-type"]?.includes("application/json")) {
    return jsonTaxCodeToApplicationTransform(response.body)!;
  }
  if (+response.status === 410 && !response.body) {
    return;
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
export interface DeleteOptions extends OperationOptions {}
export async function delete_(
  client: TaxCodesOperationsClientContext,
  taxCodeId: string,
  options?: DeleteOptions,
): Promise<void> {
  const path = parse("/{taxCodeId}").expand({
    taxCodeId: taxCodeId
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).delete(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 204 && !response.body) {
    return;
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
