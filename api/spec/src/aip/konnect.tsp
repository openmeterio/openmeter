import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";
import "./shared/index.tsp";
import "./events/index.tsp";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

/**
 * Konnect Metering & Billing API.
 */
@service(#{ title: "Konnect ${Shared.MeteringAndBillingTitle}" })
@info(#{
  version: "0.0.1",
  contact: #{ name: "Kong", url: "https://cloud.konghq.com" },
})
@server("https://us.api.konghq.com/v3", "United-States Production region")
@server("https://eu.api.konghq.com/v3", "Europe Production region")
@server("https://au.api.konghq.com/v3", "Australia Production region")
@server("https://me.api.konghq.com/v3", "Middle-East Production region")
@server("https://in.api.konghq.com/v3", "India Production region")
// TODO: Uncomment this when the Singapore production region is available
// @server("https://sg.api.konghq.com/v3", "Singapore Production region")
@server("https://global.api.konghq.com/v3", "Global Production region")
@tagMetadata(
  Shared.MeteringEventsTag,
  #{ description: Shared.MeteringEventsDescription }
)
@useAuth(systemAccountAccessToken | personalAccessToken | konnectAccessToken)
namespace MeteringAndBilling;

@route("/metering-events")
@tag(Shared.MeteringEventsTag)
@friendlyName("${Shared.MeteringAndBillingTitle}: ${Shared.MeteringEventsTag}")
interface MeteringEventsEndpoints extends Events.MeteringEventsOperations {}

/**
 * The system account access token is meant for automations and integrations that are not directly associated with a human identity.
 * You can generate a system account Access Token by creating a system account and then obtaining a system account access token for that account.
 * The access token must be passed in the header of a request, for example:
 * `curl -X GET 'https://global.api.konghq.com/v2/users/' --header 'Authorization: Bearer spat_i2Ej...'`
 */
#suppress "@openmeter/api-spec/casing-aip" "Use existing values"
@friendlyName("systemAccountAccessToken")
@useRef("../../../../common/definitions/security.yaml#/components/securitySchemes/systemAccountAccessToken")
model systemAccountAccessToken {
  #suppress "@openmeter/api-spec/doc-decorator" ""
  type: AuthType.http;
  #suppress "@openmeter/api-spec/doc-decorator" ""
  scheme: "Bearer";
}

/**
 * The personal access token is meant to be used as an alternative to basic-auth when accessing Konnect via APIs.
 * You can generate a Personal Access Token (PAT) from the [personal access token page](https://cloud.konghq.com/global/account/tokens/) in the Konnect dashboard.
 * The PAT token must be passed in the header of a request, for example:
 * `curl -X GET 'https://global.api.konghq.com/v2/users/' --header 'Authorization: Bearer kpat_xgfT...'`
 */
#suppress "@openmeter/api-spec/casing-aip" "Use existing values"
@friendlyName("personalAccessToken")
@useRef("../../../../common/definitions/security.yaml#/components/securitySchemes/personalAccessToken")
model personalAccessToken {
  #suppress "@openmeter/api-spec/doc-decorator" ""
  type: AuthType.http;
  #suppress "@openmeter/api-spec/doc-decorator" ""
  scheme: "Bearer";
}

/**
 * The Konnect access token is meant to be used by the Konnect dashboard and the decK CLI authenticate with.
 */
#suppress "@openmeter/api-spec/casing-aip" "Use existing values"
@friendlyName("konnectAccessToken")
@useRef("../../../../common/definitions/security.yaml#/components/securitySchemes/konnectAccessToken")
model konnectAccessToken {
  #suppress "@openmeter/api-spec/doc-decorator" ""
  type: AuthType.http;
  #suppress "@openmeter/api-spec/doc-decorator" ""
  scheme: "Bearer";
}
