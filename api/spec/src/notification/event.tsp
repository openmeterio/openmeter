import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.OpenAPI;

namespace OpenMeter.Notification;

/**
 * Type of the notification event.
 */
@friendlyName("NotificationEventType")
enum EventType {
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  entitlementsBalanceThreshold: "entitlements.balance.threshold",
}

/**
 * The delivery status of the notification event.
 */
@friendlyName("NotificationDeliveryStatus")
model DeliveryStatus {
  /**
   * Delivery state of the notification event to the channel.
   */
  @visibility("read")
  @summary("Delivery State")
  @example("SUCCESS")
  state: "SUCCESS" | "FAILED" | "SENDING" | "PENDING";

  /**
   * The reason of the last deliverry state update.
   */
  @visibility("read")
  @summary("State Reason")
  @example("Failed to dispatch event due to provider error.")
  reason: string;

  /**
   * Timestamp of when the status was last updated in RFC 3339 format.
   */
  @visibility("read")
  @summary("Last Update Time")
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  updatedAt: DateTime;
}

/**
 * The delivery status of the notification event.
 */
@friendlyName("NotificationEventPayload")
@discriminator("type")
@oneOf
union EventPayload {
  entitlementsBalanceThreshold: EventBalanceThresholdPayload,
}

/**
 * Payload for notification event with `entitlements.balance.threshold` type.
 */
@friendlyName("NotificationEventBalanceThresholdPayload")
model EventBalanceThresholdPayload {
  /**
   * A unique identifier for the notification event the payload belongs to.
   */
  @visibility("read")
  @summary("Notification Event Identifier")
  @example("01J2KNP1YTXQRXHTDJ4KPR7PZ0")
  id: ULID;

  /**
   * Type of the notification event.
   */
  @visibility("read")
  @summary("Notification Event Type")
  type: EventType.entitlementsBalanceThreshold;

  /**
   * Timestamp when the notification event was created in RFC 3339 format.
   */
  @visibility("read")
  @summary("Creation Time")
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  timestamp: DateTime;

  /**
   * The data of the payload.
   */
  @visibility("read")
  @summary("Payload Data")
  data: EventBalanceThresholdPayloadData;
}

/**
 * Data of the payload for notification event with `entitlements.balance.threshold` type.
 */
@friendlyName("NotificationEventBalanceThresholdPayloadData")
model EventBalanceThresholdPayloadData {
  @visibility("read")
  @summary("Entitlement")
  entitlement: OpenMeter.Entitlements.EntitlementMetered;

  @visibility("read")
  @summary("Feature")
  feature: OpenMeter.Entitlements.Feature;

  @visibility("read")
  @summary("Subject")
  subject: OpenMeter.Subject;

  @visibility("read")
  @summary("Entitlement Value")
  value: OpenMeter.Entitlements.EntitlementValue;

  @visibility("read")
  @summary("Threshold")
  threshold: RuleBalanceThresholdValue;
}

/**
 * Type of the notification event.
 */
@friendlyName("NotificationEvent")
model Event {
  /**
   * A unique identifier of the notification event.
   */
  @visibility("read")
  @summary("Event Identifier")
  @example("01J2KNP1YTXQRXHTDJ4KPR7PZ0")
  id: ULID;

  /**
   * Type of the notification event.
   */
  @visibility("read")
  @summary("Event Type")
  type: EventType;

  /**
   * Timestamp when the notification event was created in RFC 3339 format.
   */
  @visibility("read")
  @summary("Creation Time")
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  createdAt: DateTime;

  /**
   * The nnotification rule which generated this event.
   */
  @visibility("read")
  @summary("Owner Rule")
  rule: Rule;

  /**
   * The delivery status of the notification event.
   */
  @visibility("read")
  @summary("Delivery Status")
  deliveryStatus: Array<DeliveryStatus>;

  /**
   * Timestamp when the notification event was created in RFC 3339 format.
   */
  @visibility("read")
  @summary("Event Payload")
  payload: EventPayload;

  /**
   * Set of key-value pairs managed by the system. Cannot be modified by user.
   */
  @visibility("read")
  @summary("Annotations")
  annotations: Annotations;
}

/**
 * Order by options for notification channels.
 */
@friendlyName("NotificationEventOrderBy")
enum EventOrderBy {
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  id: "id",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  createdAt: "createdAt",
}

// NOTE(chrisgacsal): this is a workaround of using the model OpenMeter.PaginatedResponse<Rule>
// results a firendlyName RulePaginatedResponse instead of NotificationRulePaginatedResponse which
// seems to be a bug in typespec. This way we can override the friendlyName of the paginated response.
/**
 * Paginated response for listing notification events.
 */
@friendlyName("NotificationEventsPaginatedResponse")
model EventsPaginatedResponse is OpenMeter.PaginatedResponse<Event>;

@route("/api/v1/notification/events")
@tag("Notification (Experimental)")
interface Events {
  /**
   * List all notification events.
   */
  @get
  @operationId("listNotificationEvents")
  @summary("List notification events")
  list(
    /**
     * Start date-time in RFC 3339 format.
     * Inclusive.
     */
    @query
    @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
    from?: DateTime,

    /**
     * End date-time in RFC 3339 format.
     * Inclusive.
     */
    @query
    @example(DateTime.fromISO("2023-02-01T01:01:01.001Z"))
    to?: DateTime,

    // TODO(chrisgacsal): figure out if there is a way to use union type for attribute like this
    // where the code generation results a less complicated/more idiomatic code.
    //
    //  alias IdOrKey = ULID | Key;
    //  const features = Array<IdOrKey>;
    //

    /**
     * Filtering by multiple feature ids or keys.
     *
     * Usage: `?feature=feature-1&feature=feature-2`
     */
    @query(#{ explode: true })
    feature?: Array<string>,

    /**
     * Filtering by multiple subject ids or keys.
     *
     * Usage: `?subject=subject-1&subject=subject-2`
     */
    @query(#{ explode: true })
    subject?: Array<string>,

    /**
     * Filtering by multiple rule ids.
     *
     * Usage: `?rule=01J8J2XYZ2N5WBYK09EDZFBSZM&rule=01J8J4R4VZH180KRKQ63NB2VA5`
     */
    @query(#{ explode: true })
    rule?: Array<ULID>,

    /**
     * Filtering by multiple channel ids.
     *
     * Usage: `?channel=01J8J4RXH778XB056JS088PCYT&channel=01J8J4S1R1G9EVN62RG23A9M6J`
     */
    @query(#{ explode: true })
    channel?: Array<ULID>,

    ...OpenMeter.QueryPagination,
    ...OpenMeter.QueryOrdering<EventOrderBy>,
  ): EventsPaginatedResponse | OpenMeter.CommonErrors;

  /**
   * Get a notification event by id.
   */
  @get
  @operationId("getNotificationEvent")
  @summary("Get notification event")
  get(@path eventId: string): Event | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}
