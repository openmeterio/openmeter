import "@typespec/http";

using TypeSpec.Http;

/**
 * Paginated query parameters.
 */
@friendlyName("PaginatedQuery")
model PaginatedQuery {
  /**
   * The page number.
   */
  @query
  @minValue(1)
  page?: integer = 1;

  /**
   * The number of items in the page.
   */
  @query
  @minValue(1)
  @maxValue(1000)
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

/**
 * Paginated cursor query parameters.
 */
@friendlyName("CursorPaginatedQuery")
model CursorPaginatedQuery {
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
  @maxValue(1000)
  limit?: integer = 100;
}

/**
 * A response for cursor pagination.
 */
@friendlyName("{name}CursorList", T)
model CursorPagedList<T> {
  /**
   * The items in the response.
   */
  @pageItems
  @maxItems(1000)
  items: T[];

  /**
   * The cursor of the last item in the list.
   */
  nextCursor: string;
}
