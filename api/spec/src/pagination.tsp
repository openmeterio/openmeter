import "@typespec/http";

using TypeSpec.Http;

model PaginatedQuery {
  /**
   * The page number.
   */
  @query
  page?: integer = 1;

  /**
   * The number of items in the page.
   */
  @query
  pageSize?: integer = 100;
}

/**
 * A page of results.
 */
@friendlyName("{name}List", T)
model Paginated<T> {
  /**
   * The page number.
   */
  page: integer;

  /**
   * The number of items in the page.
   */
  @minValue(1)
  @maxValue(1000)
  pageSize: integer;

  /**
   * The total number of items.
   */
  totalCount: integer;

  /**
   * The items in the page.
   */
  @maxItems(1000)
  items: T[];
}
