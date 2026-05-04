import { parse } from "uri-template";
import type {
  FieldFiltersEndpointsClientContext,
} from "./fieldFiltersEndpointsClientContext.js";
import { createRestError } from "../../helpers/error.js";
import type { OperationOptions } from "../../helpers/interfaces.js";
import {
  jsonFieldFiltersToTransportTransform,
} from "../../models/internal/serializers.js";
import type {
  BooleanFieldFilter,
  DateTimeFieldFilter,
  FieldFilters,
  NumericFieldFilter,
  StringFieldFilter,
  StringFieldFilterExact,
  UlidFieldFilter,
} from "../../models/models.js";

export interface GetFieldFiltersOptions extends OperationOptions {
  boolean?: BooleanFieldFilter
  numeric?: NumericFieldFilter
  string?: StringFieldFilter
  stringExact?: StringFieldFilterExact
  ulid?: UlidFieldFilter
  datetime?: DateTimeFieldFilter
  labels?: Record<string, StringFieldFilter>
  filter?: FieldFilters
}
export async function getFieldFilters(
  client: FieldFiltersEndpointsClientContext,
  options?: GetFieldFiltersOptions,
): Promise<void> {
  const path = parse("/field-filters{?filter*}").expand({
    ...(options?.filter && {filter: jsonFieldFiltersToTransportTransform(options.filter)})
  });
  const httpRequestOptions = {
    headers: {},
  };
  const response = await client.pathUnchecked(path).get(httpRequestOptions);


  if (typeof options?.operationOptions?.onResponse === "function") {
    options?.operationOptions?.onResponse(response);
  }
  if (+response.status === 204 && !response.body) {
    return;
  }
  throw createRestError(response);
}
;
