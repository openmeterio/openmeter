import { parse } from "uri-template";
import type { PathUncheckedResponse } from "@typespec/ts-http-runtime";
import {
  EventsOperationsClientContext,
} from "./eventsOperationsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  buildPagedAsyncIterator,
  type PagedAsyncIterableIterator,
} from "../../helpers/pagingHelpers.js";
import {
  jsonArrayIngestedEventToApplicationTransform,
  jsonArrayMeteringEventToTransportTransform,
  jsonCursorMetaToApplicationTransform,
  jsonCursorPaginationQueryPageToTransportTransform,
  jsonListEventsParamsFilterToTransportTransform,
  jsonMeteringEventToTransportTransform,
  jsonSortQueryToTransportTransform,
} from "../../models/internal/serializers.js";
import {
  type CursorPaginationQueryPage,
  type DateTimeFieldFilter,
  IngestedEvent,
  type ListEventsParamsFilter,
  MeteringEvent,
  type SortQuery,
  type StringFieldFilter,
  type UlidFieldFilter,
} from "../../models/models.js";

export interface ListOptions extends OperationOptions {
  size?: number
  after?: string
  before?: string
  page?: CursorPaginationQueryPage
  id?: StringFieldFilter
  source?: StringFieldFilter
  subject?: StringFieldFilter
  type?: StringFieldFilter
  customerId?: UlidFieldFilter
  time?: DateTimeFieldFilter
  ingestedAt?: DateTimeFieldFilter
  storedAt?: DateTimeFieldFilter
  filter?: ListEventsParamsFilter
  sort?: SortQuery
}
export interface ListPageSettings {}
export interface ListPageResponse {
  data: Array<IngestedEvent>
}
async function listSend(
  client: EventsOperationsClientContext,
  options?: Record<string, any>,
) {
  const path = parse("/{?page*,filter*,sort}").expand({
    ...(options?.page && {page: jsonCursorPaginationQueryPageToTransportTransform(options.page)}),
    ...(options?.filter && {filter: jsonListEventsParamsFilterToTransportTransform(options.filter)}),
    ...(options?.sort && {sort: jsonSortQueryToTransportTransform(options.sort)})
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
      data: jsonArrayIngestedEventToApplicationTransform(response.body.data),meta: jsonCursorMetaToApplicationTransform(response.body.meta)
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
  client: EventsOperationsClientContext,
  options?: ListOptions,
): PagedAsyncIterableIterator<IngestedEvent,ListPageResponse,ListPageSettings> {
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
  return buildPagedAsyncIterator<IngestedEvent, ListPageResponse, ListPageSettings>({getElements, getPagedResponse: getPagedResponse as any});
}
export interface IngestEventOptions extends OperationOptions {
  contentType?: "application/cloudevents+json"
}
/**
 * Ingests an event or batch of events following the CloudEvents specification.
 *
 * @param {EventsOperationsClientContext} client
 * @param {MeteringEvent} body
 * @param {IngestEventOptions} [options]
 */
export async function ingestEvent(
  client: EventsOperationsClientContext,
  body: MeteringEvent,
  options?: IngestEventOptions,
): Promise<void> {
  const path = parse("/").expand({});
  const httpRequestOptions = {
    headers: {
      "content-type": options?.contentType ?? "application/cloudevents+json"
    },body: jsonMeteringEventToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 202 && !response.body) {
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
export interface IngestEventsOptions extends OperationOptions {
  contentType?: "application/cloudevents-batch+json"
}
export async function ingestEvents(
  client: EventsOperationsClientContext,
  body: Array<MeteringEvent>,
  options?: IngestEventsOptions,
): Promise<void> {
  const path = parse("/").expand({});
  const httpRequestOptions = {
    headers: {
      "content-type": options?.contentType ?? "application/cloudevents-batch+json"
    },body: jsonArrayMeteringEventToTransportTransform(body),
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 202 && !response.body) {
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
export interface IngestEventsJsonOptions extends OperationOptions {
  contentType?: "application/json"
}
export async function ingestEventsJson(
  client: EventsOperationsClientContext,
  body: MeteringEvent | Array<MeteringEvent>,
  options?: IngestEventsJsonOptions,
): Promise<void> {
  const path = parse("/").expand({});
  const httpRequestOptions = {
    headers: {
      "content-type": options?.contentType ?? "application/json"
    },body: body,
  };
  const response = await client.pathUnchecked(path).post(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 202 && !response.body) {
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
