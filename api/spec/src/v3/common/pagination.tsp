import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Common;

/**
 * Cursor page query.
 */
@friendlyName("CursorPageQuery")
model CursorPageQuery {
  /**
   * Determines which page of the collection to retrieve.
   */
  #suppress "@openmeter/api-spec/casing-aip" "common model"
  @query
  @useRef("../../../../common/definitions/metadatas.yaml#/components/parameters/CursorPageQuery")
  page?: {
    size?: integer;
    after?: string;
    before?: string;
  };
}

/**
 * Cursor pagination metadata.
 */
@friendlyName("CursorMetaPage")
@useRef("../../../../common/definitions/metadatas.yaml#/components/schemas/CursorMetaPage")
model CursorMetaPage {
  /**
   * URI to the first page.
   */
  @format("uri")
  first?: string;

  /**
   * URI to the last page.
   */
  @format("uri")
  last?: string;

  /**
   * URI to the next page.
   */
  @format("uri")
  next?: string;

  /**
   * URI to the previous page.
   */
  @format("uri")
  previous?: string;

  /**
   * Requested page size.
   */
  size?: integer;
}

/**
 * Cursor pagination metadata.
 */
@friendlyName("CursorMeta")
@useRef("../../../../common/definitions/metadatas.yaml#/components/schemas/CursorMeta")
model CursorMeta {
  /**
   * Page metadata.
   */
  page: CursorMetaPage;
}

/**
 * Cursor pagination metadata with total.
 */
@friendlyName("CursorMetaWithTotal")
@useRef("../../../../common/definitions/metadatas.yaml#/components/schemas/CursorMetaWithTotal")
model CursorMetaWithTotal {
  ...CursorMeta;

  /**
   * Total number of items in the collection.
   */
  total: integer;
}

/**
 * Cursor pagination metadata with estimated total.
 */
@friendlyName("CursorMetaWithEstimatedTotal")
@useRef("../../../../common/definitions/metadatas.yaml#/components/schemas/CursorMetaWithEstimatedTotal")
model CursorMetaWithEstimatedTotal {
  ...CursorMeta;

  /**
   * Estimated total number of items in the collection.
   */
  estimated_total: integer;
}

/**
 * Page before.
 */
@friendlyName("PageBefore")
model PageBefore {
  /**
   * Request the next page of data, starting with the item before this parameter.
   */
  #suppress "@openmeter/api-spec/casing-aip" "common model"
  @query
  @example("ewogICJpZCI6ICJoZWxsbyB3b3JsZCIKfQ")
  @useRef("../../../../common/definitions/metadatas.yaml#/components/parameters/PageBefore")
  `page[before]`?: string;
}

/**
 * Page after.
 */
@friendlyName("PageAfter")
model PageAfter {
  /**
   * Request the next page of data, starting with the item after this parameter.
   */
  #suppress "@openmeter/api-spec/casing-aip" "common model"
  @query
  @example("ewogICJpZCI6ICJoZWxsbyB3b3JsZCIKfQ")
  @useRef("../../../../common/definitions/metadatas.yaml#/components/parameters/PageAfter")
  `page[after]`?: string;
}

/**
 * Page size.
 */
@friendlyName("PageSize")
model PageSize {
  /**
   * The maximum number of items to include per page. The last page of a collection may include fewer items.
   */
  #suppress "@openmeter/api-spec/casing-aip" "common model"
  @query
  @example(10)
  @useRef("../../../../common/definitions/metadatas.yaml#/components/parameters/PageSize")
  `page[size]`?: integer;
}

/**
 * Page number.
 */
@friendlyName("PageNumber")
model PageNumber {
  /**
   * Determines which page of the entities to retrieve.
   */
  #suppress "@openmeter/api-spec/casing-aip" "common model"
  @query
  @example(1)
  @useRef("../../../../common/definitions/metadatas.yaml#/components/parameters/PageNumber")
  `page[number]`?: integer;
}

/**
 * Pagination metadata.
 */
@friendlyName("PaginatedMeta")
@useRef("../../../../common/definitions/metadatas.yaml#/components/schemas/PaginatedMeta")
model PaginatedMeta {
  /**
   * Page metadata.
   */
  page: PageMeta;
}

/**
 * Pagination information.
 */
@friendlyName("PageMeta")
@useRef("../../../../common/definitions/metadatas.yaml#/components/schemas/PageMeta")
model PageMeta {
  /**
   * Page number.
   */
  number: integer;

  /**
   * Page size.
   */
  size: integer;

  /**
   * Total number of items in the collection.
   */
  total: integer;
}
