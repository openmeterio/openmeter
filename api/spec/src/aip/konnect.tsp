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
@server("https://global.api.konghq.com/v3", "Global Production region")
@tagMetadata(Shared.MetersTag, #{ description: Shared.MetersDescription })
@tagMetadata(
  Shared.UsageEventsTag,
  #{ description: Shared.UsageEventsDescription }
)
namespace MeteringAndBilling;

@route("/events")
@tag(Shared.UsageEventsTag)
@friendlyName("${Shared.MeteringAndBillingTitle}: ${Shared.UsageEventsTag}")
interface UsageEventsEndpoints extends Events.UsageEventsOperations {}
