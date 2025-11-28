import "../common/pagination.tsp";

namespace Shared;

/**
 * Cursor paginated response.
 */
@friendlyName("{name}PaginatedResponse", T)
model CursorPaginatedResponse<T> {
  @pageItems
  data: T[];

  meta: Common.CursorMeta;
}
