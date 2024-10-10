import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace Invoice.CBC;

@pattern("^(?:[a-z]|[a-z0-9][a-z0-9-+]*[a-z0-9])$")
@minLength(1)
@maxLength(64)
scalar Key extends string;

@pattern("^(?:[a-z]|[a-z0-9][a-z0-9-+]*[a-z0-9])$")
scalar MetaKey extends string;

model Meta {
  ...Record<MetaKey>;
}

@pattern("^[A-Za-z0-9]+([\\.\\-\\/ _]?[A-Za-z0-9]+)*$")
@minLength(1)
@maxLength(32)
scalar Code extends string;
