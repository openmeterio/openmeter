import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;

namespace OpenMeter;

/**
 * Pagination query params
 */
@friendlyName("Pagination")
model QueryPagination {
  /**
   * Page index.
   *
   * Default is 1.
   */
  @query
  @minValue(1)
  @example(1)
  @pageIndex
  page?: integer = 1;

  /**
   * The maximum number of items per page.
   *
   * Default is 100.
   */
  @query
  @minValue(1)
  @maxValue(1000)
  @example(100)
  @pageSize
  pageSize?: integer = 100;
}

/**
 * Limit and offset query params
 */
@friendlyName("LimitOffset")
model QueryLimitOffset {
  /**
   * Number of items to skip.
   *
   * Default is 0.
   */
  @query
  @minValue(0)
  @example(0)
  offset?: integer = 0;

  /**
   * Number of items to return.
   *
   * Default is 100.
   */
  @query
  @minValue(1)
  @maxValue(1000)
  @example(100)
  limit?: integer = 100;
}

/**
 * Ordering query params
 */
@friendlyName("{name}Ordering", T)
model QueryOrdering<T> {
  /**
   * The order direction.
   */
  @query
  @example(SortOrder.ASC)
  order?: SortOrder = SortOrder.ASC;

  /**
   * The order by field.
   *
   * @TODO Default values cannot be passed as generics in typespec,
   * even though the docs say so:
   * https://typespec.io/docs/language-basics/templates
   */
  @query
  orderBy?: T;
}

/**
 * The order direction.
 */
@friendlyName("SortOrder")
enum SortOrder {
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  ASC: "ASC",
  #suppress "@openmeter/api-spec/casing" "Use existing values"
  DESC: "DESC",
}

/**
 * Paginated response
 */
@friendlyName("{name}PaginatedResponse", T)
model PaginatedResponse<T> {
  /**
   * The total number of items.
   */
  @example(500)
  totalCount: integer;

  /**
   * The page index.
   */
  @example(1)
  page: integer;

  /**
   * The maximum number of items per page.
   */
  @example(100)
  pageSize: integer;

  /**
   * The items in the current page.
   */
  @pageItems
  items: T[];
}

/**
 * Paginated cursor query parameters.
 */
@friendlyName("CursorPagination")
model QueryCursorPagination {
  /**
   * The cursor after which to start the pagination.
   */
  @query
  cursor?: string;

  /**
   * The limit of the pagination.
   */
  @query
  @minValue(1)
  @maxValue(100)
  limit?: integer = 100;
}

/**
 * A response for cursor pagination.
 */
@friendlyName("{name}CursorPaginatedResponse", T)
model CursorPaginatedResponse<T> {
  /**
   * The items in the response.
   */
  @pageItems
  @maxItems(100)
  items: T[];

  /**
   * The cursor of the last item in the list.
   */
  nextCursor?: string;
}
