import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace Common;

/**
 * Determines which page of the collection to retrieve.
 */
@friendlyName("CursorPaginationQueryPage")
model CursorPaginationQueryPage {
  /**
   * Maximum number of customer rows to include in the response.
   */
  size?: integer;

  /**
   * Request the next page of data, starting with the item after this parameter.
   */
  after?: string;

  /**
   * Request the previous page of data, starting with the item before this parameter.
   */
  before?: string;
}

/**
 * Cursor page query.
 */
@friendlyName("CursorPaginationQuery")
model CursorPaginationQuery {
  #suppress "@openmeter/api-spec-aip/doc-decorator" "-"
  @query(#{ explode: true, style: "deepObject" })
  page?: CursorPaginationQueryPage;
}

/**
 * Page pagination query.
 */
@friendlyName("PagePaginationQuery")
model PagePaginationQuery {
  /**
   * Determines which page of the collection to retrieve.
   */
  @query(#{ explode: true, style: "deepObject" })
  page?: {
    /**
     * The number of items to include per page.
     */
    size?: integer;

    /**
     * The page number.
     */
    number?: integer;
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
 * Pagination metadata.
 */
@friendlyName("PaginatedMeta")
@useRef("../../../../common/definitions/metadatas.yaml#/components/schemas/PaginatedMeta")
model PageMeta {
  /**
   * Page metadata.
   */
  page: PagePaginatedMeta;
}

/**
 * Pagination information.
 */
@friendlyName("PageMeta")
@useRef("../../../../common/definitions/metadatas.yaml#/components/schemas/PageMeta")
model PagePaginatedMeta {
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
