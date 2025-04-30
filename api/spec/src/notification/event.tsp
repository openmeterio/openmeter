import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace OpenMeter.Notification;

/**
 * Type of the notification event.
 */
@friendlyName("NotificationEventType")
enum NotificationEventType {
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  entitlementsBalanceThreshold: "entitlements.balance.threshold",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  invoiceCreated: "invoice.created",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  invoiceUpdated: "invoice.updated",
}

/**
 * The delivery status of the notification event.
 */
@friendlyName("NotificationEventDeliveryStatus")
model NotificationEventDeliveryStatus {
  /**
   * Delivery state of the notification event to the channel.
   */
  @visibility(Lifecycle.Read)
  @example(NotificationEventDeliveryStatusState.Success)
  state: NotificationEventDeliveryStatusState;

  /**
   * The reason of the last deliverry state update.
   */
  @visibility(Lifecycle.Read)
  @summary("State Reason")
  @example("Failed to dispatch event due to provider error.")
  reason: string;

  /**
   * Timestamp of when the status was last updated in RFC 3339 format.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  updatedAt: DateTime;

  /**
   * Notification channel the delivery sattus associated with.
   */
  @visibility(Lifecycle.Read)
  @summary("Notification Channel")
  channel: NotificationChannelMeta;
}

/**
 * The delivery state of the notification event to the channel.
 */
@friendlyName("NotificationEventDeliveryStatusState")
@summary("Delivery State")
@extension("x-enum-varnames", #["Success", "Failed", "Sending", "Pending"])
enum NotificationEventDeliveryStatusState {
  Success: "SUCCESS",
  Failed: "FAILED",
  Sending: "SENDING",
  Pending: "PENDING",
}

/**
 * The delivery status of the notification event.
 */
@friendlyName("NotificationEventPayload")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union NotificationEventPayload {
  `entitlements.balance.threshold`: NotificationEventBalanceThresholdPayload,
  `invoice.created`: NotificationEventInvoiceCreatedPayload,
  `invoice.updated`: NotificationEventInvoiceUpdatedPayload,
}

/**
 * Type of the notification event.
 */
@friendlyName("NotificationEvent")
model NotificationEvent {
  /**
   * A unique identifier of the notification event.
   */
  @visibility(Lifecycle.Read)
  @summary("Event Identifier")
  @example("01J2KNP1YTXQRXHTDJ4KPR7PZ0")
  id: ULID;

  /**
   * Type of the notification event.
   */
  @visibility(Lifecycle.Read)
  @summary("Event Type")
  type: NotificationEventType;

  /**
   * Timestamp when the notification event was created in RFC 3339 format.
   */
  @visibility(Lifecycle.Read)
  @summary("Creation Time")
  @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
  createdAt: DateTime;

  /**
   * The nnotification rule which generated this event.
   */
  @visibility(Lifecycle.Read)
  rule: NotificationRule;

  /**
   * The delivery status of the notification event.
   */
  @visibility(Lifecycle.Read)
  @summary("Delivery Status")
  deliveryStatus: Array<NotificationEventDeliveryStatus>;

  /**
   * Timestamp when the notification event was created in RFC 3339 format.
   */
  @visibility(Lifecycle.Read)
  payload: NotificationEventPayload;

  /**
   * Set of key-value pairs managed by the system. Cannot be modified by user.
   */
  @visibility(Lifecycle.Read)
  @summary("Annotations")
  annotations?: Annotations;
}

/**
 * Order by options for notification channels.
 */
@friendlyName("NotificationEventOrderBy")
enum NotificationEventOrderBy {
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  id: "id",
  #suppress "@openmeter/api-spec/casing" "Ignore due to backward compatibility"
  createdAt: "createdAt",
}

@route("/api/v1/notification/events")
@tag("Notifications")
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
    @query(#{ explode: true })
    @example(DateTime.fromISO("2023-01-01T01:01:01.001Z"))
    from?: DateTime,

    /**
     * End date-time in RFC 3339 format.
     * Inclusive.
     */
    @query(#{ explode: true })
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
    ...OpenMeter.QueryOrdering<NotificationEventOrderBy>,
  ): OpenMeter.PaginatedResponse<NotificationEvent> | OpenMeter.CommonErrors;

  /**
   * Get a notification event by id.
   */
  @get
  @operationId("getNotificationEvent")
  @summary("Get notification event")
  get(
    @path eventId: string,
  ): NotificationEvent | OpenMeter.NotFoundError | OpenMeter.CommonErrors;
}
