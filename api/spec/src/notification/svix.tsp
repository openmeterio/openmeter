import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Notification;

/**
 * Operational webhook reqeuest sent by Svix.
 */
@friendlyName("SvixOperationalWebhookRequest")
model SvixOperationalWebhookRequest {
  /**
   * The type of the Svix operational webhook request.
   */
  @summary("Operational Webhook Type")
  type:
    | "endpoint.created"
    | "endpoint.deleted"
    | "endpoint.disabled"
    | "endpoint.updated"
    | "message.attempt.exhausted"
    | "message.attempt.failing"
    | "message.attempt.recovered";

  /**
   * The payload of the Svix operational webhook request.
   */
  @summary("Operational Webhook Payload")
  data: Record<string>;
}

@route("/api/v1/notification/webhook/svix")
@tag("Notification (Experimental)")
interface SvixOperationalWebhooks {
  /**
   * Callback endpoint used by Svix to notify about operational events.
   */
  @post
  @operationId("receiveSvixOperationalEvent")
  @summary("Receive Svix operational events")
  create(@body request: SvixOperationalWebhookRequest): {
    @statusCode _: 204;
  } | OpenMeter.CommonErrors;
}
