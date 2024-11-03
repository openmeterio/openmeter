import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter;

/**
 * Pagination query params
 */
@friendlyName("Pagination")
model QueryPagination {
  /**
   * Start date-time in RFC 3339 format.
   *
   * Inclusive.
   */
  @query
  @minValue(1)
  @example(1)
  page?: integer = 1;

  /**
   * Number of items per page.
   *
   * Default is 100.
   */
  @query
  @minValue(1)
  @maxValue(1000)
  @example(100)
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
   * The items in the current page.
   */
  @example(500)
  totalCount: integer;

  /**
   * The items in the current page.
   */
  @example(1)
  page: integer;

  /**
   * The items in the current page.
   */
  @example(100)
  pageSize: integer;

  /**
   * The items in the current page.
   */
  items: T[];
}
