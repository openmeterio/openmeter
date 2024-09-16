import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

// TODO: does this have to be in a separate namespace?
namespace OpenMeter.Entitlements;

@route("/api/v1/entitlements")
@tag("Entitlements (Experimental)")
interface Entitlements {

}