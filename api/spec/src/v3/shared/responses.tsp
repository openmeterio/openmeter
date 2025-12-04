import "@typespec/rest";
import "@typespec/http";
import "../common/properties.tsp";

namespace Shared;

@doc("{name} response.", T)
@friendlyName("Get{name}Response", T)
model GetResponse<T extends {}> {
  @Http.statusCode _: 200;
  @Http.body body: T;
}

@doc("{name} created response.", T)
@friendlyName("Create{name}Response", T)
model CreateResponse<T extends {}> {
  @Http.statusCode _: 201;
  @Http.body body: T;
}

@doc("Cursor paginated response.")
@friendlyName("{name}PaginatedResponse", T)
model CursorPaginatedResponse<T> {
  @Http.statusCode _: 200;

  @pageItems
  data: T[];

  meta: Common.CursorMeta;
}
