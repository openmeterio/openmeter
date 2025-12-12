import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

namespace Common;

/**
 * JSON primitive.
 */
alias JsonPrimitive = string | int32 | float64 | boolean | null;

/**
 * JSON value.
 */
@friendlyName("JsonValue")
union JsonValue {
  JsonPrimitive,
  JsonObject,
  JsonArray,
}

/**
 * JSON object.
 */
@friendlyName("JsonObject")
model JsonObject is Record<JsonValue>;

/**
 * JSON array (named type for JsonValue[]).
 */
@friendlyName("JsonArray")
model JsonArray is JsonValue[];
