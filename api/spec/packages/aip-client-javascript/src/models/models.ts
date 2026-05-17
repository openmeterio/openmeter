/**
 * A sequence of textual characters.
 */
export type String = string;
/**
 * Determines which page of the collection to retrieve.
 */
export interface CursorPaginationQueryPage {
  /**
   * The number of items to include per page.
   */
  size?: number;
  /**
   * Request the next page of data, starting with the item after this parameter.
   */
  after?: string;
  /**
   * Request the previous page of data, starting with the item before this parameter.
   */
  before?: string;
}
/**
 * A whole number. This represent any `integer` value possible.
 * It is commonly represented as `BigInteger` in some languages.
 */
export type Integer = number;
/**
 * A numeric type
 */
export type Numeric = number;
/**
 * Filter options for listing ingested events.
 */
export interface ListEventsParamsFilter {
  /**
   * Filter events by ID.
   */
  id?: StringFieldFilter;
  /**
   * Filter events by source.
   */
  source?: StringFieldFilter;
  /**
   * Filter events by subject.
   */
  subject?: StringFieldFilter;
  /**
   * Filter events by type.
   */
  type?: StringFieldFilter;
  /**
   * Filter events by the associated customer ID.
   */
  customerId?: UlidFieldFilter;
  /**
   * Filter events by event time.
   */
  time?: DateTimeFieldFilter;
  /**
   * Filter events by the time the event was ingested.
   */
  ingestedAt?: DateTimeFieldFilter;
  /**
   * Filter events by the time the event was stored.
   */
  storedAt?: DateTimeFieldFilter;
}
/**
 * Filters on the given string field value by either exact or fuzzy match. All
 * properties are optional; provide exactly one to specify the comparison.
 */
export type StringFieldFilter = string | {
  /**
   * Value strictly equals the given string value.
   */
  eq?: string;
  /**
   * Value does not equal the given string value.
   */
  neq?: string;
  /**
   * Value contains the given string value (fuzzy match).
   */
  contains?: string;
  /**
   * Returns entities that fuzzy-match any of the comma-delimited phrases in the
   * filter string.
   */
  ocontains?: Array<string>;
  /**
   * Returns entities that exact match any of the comma-delimited phrases in the
   * filter string.
   */
  oeq?: Array<string>;
  /**
   * Value is greater than the given string value (lexicographic compare).
   */
  gt?: string;
  /**
   * Value is greater than or equal to the given string value (lexicographic
   * compare).
   */
  gte?: string;
  /**
   * Value is less than the given string value (lexicographic compare).
   */
  lt?: string;
  /**
   * Value is less than or equal to the given string value (lexicographic compare).
   */
  lte?: string;
  /**
   * When true, the field must be present (non-null); when false, the field must be
   * absent (null).
   */
  exists?: boolean;
};
/**
 * Boolean with `true` and `false` values.
 */
export type Boolean = boolean;
/**
 * Filters on the given ULID field value by exact match. All properties are
 * optional; provide exactly one to specify the comparison.
 */
export type UlidFieldFilter = string | {
  /**
   * Value strictly equals the given ULID value.
   */
  eq?: string;
  /**
   * Returns entities that exact match any of the comma-delimited ULIDs in the filter
   * string.
   */
  oeq?: Array<string>;
  /**
   * Value does not equal the given ULID value.
   */
  neq?: string;
};
/**
 * ULID (Universally Unique Lexicographically Sortable Identifier).
 */
export type Ulid = string;
/**
 * Filters on the given datetime (RFC-3339) field value. All properties are
 * optional; provide exactly one to specify the comparison.
 */
export type DateTimeFieldFilter = Date | {
  /**
   * Value strictly equals given RFC-3339 formatted timestamp in UTC.
   */
  eq?: Date;
  /**
   * Value is less than the given RFC-3339 formatted timestamp in UTC.
   */
  lt?: Date;
  /**
   * Value is less than or equal to the given RFC-3339 formatted timestamp in UTC.
   */
  lte?: Date;
  /**
   * Value is greater than the given RFC-3339 formatted timestamp in UTC.
   */
  gt?: Date;
  /**
   * Value is greater than or equal to the given RFC-3339 formatted timestamp in UTC.
   */
  gte?: Date;
};
/**
 * [RFC3339](https://tools.ietf.org/html/rfc3339) formatted date-time string in
 * UTC.
 */
export type DateTime = Date;
/**
 * An instant in coordinated universal time (UTC)"
 */
export type UtcDateTime = Date;
/**
 * Sort query.
 *
 * The `asc` suffix is optional as the default sort order is ascending. The `desc`
 * suffix is used to specify a descending order.
 */
export interface SortQuery {}
/**
 * An ingested metering event with ingestion metadata.
 */
export interface IngestedEvent {
  /**
   * The original event ingested.
   */
  event: MeteringEvent;
  /**
   * The customer if the event is associated with a customer.
   */
  customer?: ResourceReference;
  /**
   * The date and time the event was ingested and its processing started.
   */
  ingestedAt: Date;
  /**
   * The date and time the event was stored in the database.
   */
  storedAt: Date;
  /**
   * The validation errors of the ingested event.
   */
  validationErrors?: Array<IngestedEventValidationError>;
}
/**
 * Metering event following the CloudEvents specification.
 */
export interface MeteringEvent {
  /**
   * Identifies the event.
   */
  id: string;
  /**
   * Identifies the context in which an event happened.
   */
  source: string;
  /**
   * The version of the CloudEvents specification which the event uses.
   */
  specversion: string;
  /**
   * Contains a value describing the type of event related to the originating
   * occurrence.
   */
  type: string;
  /**
   * Content type of the CloudEvents data value. Only the value "application/json" is
   * allowed over HTTP.
   */
  datacontenttype?: "application/json" | null;
  /**
   * Identifies the schema that data adheres to.
   */
  dataschema?: string | null;
  /**
   * Describes the subject of the event in the context of the event producer
   * (identified by source).
   */
  subject: string;
  /**
   * Timestamp of when the occurrence happened. Must adhere to RFC 3339.
   */
  time?: Date | null;
  /**
   * The event payload. Optional, if present it must be a JSON object.
   */
  data?: Record<string, unknown> | null;
}
/**
 * Represent a URL string as described by https://url.spec.whatwg.org/
 */
export type Url = string;
/**
 * Customer reference.
 */
export interface ResourceReference {
  id: string;
}
/**
 * Event validation errors.
 */
export interface IngestedEventValidationError {
  /**
   * The machine readable code of the error.
   */
  code: string;
  /**
   * The human readable description of the error.
   */
  message: string;
  /**
   * Additional attributes.
   */
  attributes?: Record<string, unknown>;
}
/**
 * Cursor pagination metadata.
 */
export interface CursorMeta {
  /**
   * Page metadata.
   */
  page: CursorMetaPage;
}
/**
 * Cursor pagination metadata.
 */
export interface CursorMetaPage {
  /**
   * URI to the first page.
   */
  first?: string;
  /**
   * URI to the last page.
   */
  last?: string;
  /**
   * URI to the next page.
   */
  next?: string;
  /**
   * URI to the previous page.
   */
  previous?: string;
  /**
   * Requested page size.
   */
  size?: number;
}
/**
 * Meter create request.
 */
export interface CreateRequest {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  key: string;
  /**
   * The aggregation type to use for the meter.
   */
  aggregation: MeterAggregation;
  /**
   * The event type to include in the aggregation.
   */
  eventType: string;
  /**
   * The date since the meter should include events. Useful to skip old events. If
   * not specified, all historical events are included.
   */
  eventsFrom?: Date;
  /**
   * JSONPath expression to extract the value from the ingested event's data
   * property.
   *
   * The ingested value for sum, avg, min, and max aggregations is a number or a
   * string that can be parsed to a number.
   *
   * For unique_count aggregation, the ingested value must be a string. For count
   * aggregation the value_property is ignored.
   */
  valueProperty?: string;
  /**
   * Named JSONPath expressions to extract the group by values from the event data.
   *
   * Keys must be unique and consist only alphanumeric and underscore characters.
   */
  dimensions?: Record<string, string>;
}
/**
 * Labels store metadata of an entity that can be used for filtering an entity list
 * or for searching across entity types.
 *
 * Keys must be of length 1-63 characters, and cannot start with "kong", "konnect",
 * "mesh", "kic", or "\_".
 */
export interface Labels {}
/**
 * A key is a unique string that is used to identify a resource.
 */
export type ResourceKey = string;
/**
 * The aggregation type to use for the meter.
 */
export type MeterAggregation = "sum" | "count" | "unique_count" | "avg" | "min" | "max" | "latest";
/**
 * A meter is a configuration that defines how to match and aggregate events.
 */
export interface Meter {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  key: string;
  /**
   * The aggregation type to use for the meter.
   */
  aggregation: MeterAggregation;
  /**
   * The event type to include in the aggregation.
   */
  eventType: string;
  /**
   * The date since the meter should include events. Useful to skip old events. If
   * not specified, all historical events are included.
   */
  eventsFrom?: Date;
  /**
   * JSONPath expression to extract the value from the ingested event's data
   * property.
   *
   * The ingested value for sum, avg, min, and max aggregations is a number or a
   * string that can be parsed to a number.
   *
   * For unique_count aggregation, the ingested value must be a string. For count
   * aggregation the value_property is ignored.
   */
  valueProperty?: string;
  /**
   * Named JSONPath expressions to extract the group by values from the event data.
   *
   * Keys must be unique and consist only alphanumeric and underscore characters.
   */
  dimensions?: Record<string, string>;
}
/**
 * Filter options for listing meters.
 */
export interface ListMetersParamsFilter {
  /**
   * Filter meters by key.
   */
  key?: StringFieldFilter;
  /**
   * Filter meters by name.
   */
  name?: StringFieldFilter;
}
/**
 * Pagination metadata.
 */
export interface PageMeta {
  /**
   * Page metadata.
   */
  page: PagePaginatedMeta;
}
/**
 * Pagination information.
 */
export interface PagePaginatedMeta {
  /**
   * Page number.
   */
  number: number;
  /**
   * Page size.
   */
  size: number;
  /**
   * Total number of items in the collection.
   */
  total: number;
}
/**
 * Meter update request.
 */
export interface UpdateRequest {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name?: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * Named JSONPath expressions to extract the group by values from the event data.
   *
   * Keys must be unique and consist only alphanumeric and underscore characters.
   */
  dimensions?: Record<string, string>;
}
/**
 * A meter query request.
 */
export interface MeterQueryRequest {
  /**
   * The start of the period the usage is queried from.
   */
  from?: Date;
  /**
   * The end of the period the usage is queried to.
   */
  to?: Date;
  /**
   * The size of the time buckets to group the usage into. If not specified, the
   * usage is aggregated over the entire period.
   */
  granularity?: MeterQueryGranularity;
  /**
   * The value is the name of the time zone as defined in the IANA Time Zone Database
   * (http://www.iana.org/time-zones). The time zone is used to determine the start
   * and end of the time buckets. If not specified, the UTC timezone will be used.
   */
  timeZone?: string;
  /**
   * The dimensions to group the results by.
   */
  groupByDimensions?: Array<string>;
  /**
   * Filters to apply to the query.
   */
  filters?: MeterQueryFilters;
}
/**
 * The granularity of the time grouping. Time durations are specified in ISO 8601
 * format.
 */
export type MeterQueryGranularity = "PT1M" | "PT1H" | "P1D" | "P1M";
/**
 * Filters to apply to a meter query.
 */
export interface MeterQueryFilters {
  /**
   * Filters to apply to the dimensions of the query. For `subject` and `customer_id`
   * only equals ("eq", "in") comparisons are supported.
   */
  dimensions?: Record<string, QueryFilterStringMapItem>;
}
/**
 * A query filter for an item in a string map attribute. Operators are mutually
 * exclusive, only one operator is allowed at a time.
 */
export interface QueryFilterStringMapItem {
  /**
   * The attribute exists.
   */
  exists?: boolean;
  /**
   * The attribute equals the provided value.
   */
  eq?: string;
  /**
   * The attribute does not equal the provided value.
   */
  neq?: string;
  /**
   * The attribute is one of the provided values.
   */
  in_?: Array<string>;
  /**
   * The attribute is not one of the provided values.
   */
  nin?: Array<string>;
  /**
   * The attribute contains the provided value.
   */
  contains?: string;
  /**
   * The attribute does not contain the provided value.
   */
  ncontains?: string;
  /**
   * Combines the provided filters with a logical AND.
   */
  and?: Array<QueryFilterString>;
  /**
   * Combines the provided filters with a logical OR.
   */
  or?: Array<QueryFilterString>;
}
/**
 * A query filter for a string attribute. Operators are mutually exclusive, only
 * one operator is allowed at a time.
 */
export interface QueryFilterString {
  /**
   * The attribute equals the provided value.
   */
  eq?: string;
  /**
   * The attribute does not equal the provided value.
   */
  neq?: string;
  /**
   * The attribute is one of the provided values.
   */
  in_?: Array<string>;
  /**
   * The attribute is not one of the provided values.
   */
  nin?: Array<string>;
  /**
   * The attribute contains the provided value.
   */
  contains?: string;
  /**
   * The attribute does not contain the provided value.
   */
  ncontains?: string;
  /**
   * Combines the provided filters with a logical AND.
   */
  and?: Array<QueryFilterString>;
  /**
   * Combines the provided filters with a logical OR.
   */
  or?: Array<QueryFilterString>;
}
/**
 * Meter query result.
 */
export interface MeterQueryResult {
  /**
   * The start of the period the usage is queried from.
   */
  from?: Date;
  /**
   * The end of the period the usage is queried to.
   */
  to?: Date;
  /**
   * The usage data. If no data is available, an empty array is returned.
   */
  data: Array<MeterQueryRow>;
}
/**
 * A row in the result of a meter query.
 */
export interface MeterQueryRow {
  /**
   * The aggregated value.
   */
  value: string;
  /**
   * The start of the time bucket the value is aggregated over.
   */
  from: Date;
  /**
   * The end of the time bucket the value is aggregated over.
   */
  to: Date;
  /**
   * The dimensions the value is aggregated over. `subject` and `customer_id` are
   * reserved dimensions.
   */
  dimensions: Record<string, string>;
}
/**
 * Numeric represents an arbitrary precision number.
 */
export type Numeric_2 = string;
/**
 * Customer create request.
 */
export interface CreateRequest_2 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  key: string;
  /**
   * Mapping to attribute metered usage to the customer by the event subject.
   */
  usageAttribution?: CustomerUsageAttribution;
  /**
   * The primary email address of the customer.
   */
  primaryEmail?: string;
  /**
   * Currency of the customer. Used for billing, tax and invoicing.
   */
  currency?: string;
  /**
   * The billing address of the customer. Used for tax and invoicing.
   */
  billingAddress?: Address;
}
/**
 * ExternalResourceKey is a unique string that is used to identify a resource in an
 * external system.
 */
export type ExternalResourceKey = string;
/**
 * Mapping to attribute metered usage to the customer. One customer can have zero
 * or more subjects, but one subject can only belong to one customer.
 */
export interface CustomerUsageAttribution {
  /**
   * The subjects that are attributed to the customer. Can be empty when no usage
   * event subjects are associated with the customer.
   */
  subjectKeys: Array<string>;
}
/**
 * Subject key.
 */
export type UsageAttributionSubjectKey = string;
/**
 * Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html)
 * currency code. Custom three-letter currency codes are also supported for
 * convenience.
 */
export type CurrencyCode = string;
/**
 * Address
 */
export interface Address {
  /**
   * Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html)
   * alpha-2 format.
   */
  country?: string;
  /**
   * Postal code.
   */
  postalCode?: string;
  /**
   * State or province.
   */
  state?: string;
  /**
   * City.
   */
  city?: string;
  /**
   * First line of the address.
   */
  line1?: string;
  /**
   * Second line of the address.
   */
  line2?: string;
  /**
   * Phone number.
   */
  phoneNumber?: string;
}
/**
 * [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country
 * code. Custom two-letter country codes are also supported for convenience.
 */
export type CountryCode = string;
/**
 * Customers can be individuals or organizations that can subscribe to plans and
 * have access to features.
 */
export interface Customer {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  key: string;
  /**
   * Mapping to attribute metered usage to the customer by the event subject.
   */
  usageAttribution?: CustomerUsageAttribution;
  /**
   * The primary email address of the customer.
   */
  primaryEmail?: string;
  /**
   * Currency of the customer. Used for billing, tax and invoicing.
   */
  currency?: string;
  /**
   * The billing address of the customer. Used for tax and invoicing.
   */
  billingAddress?: Address;
}
/**
 * Filter options for listing customers.
 */
export interface ListCustomersParamsFilter {
  key?: StringFieldFilter;
  name?: StringFieldFilter;
  primaryEmail?: StringFieldFilter;
  usageAttributionSubjectKey?: StringFieldFilter;
  planKey?: StringFieldFilter;
  billingProfileId?: UlidFieldFilter;
}
/**
 * Customer upsert request.
 */
export interface UpsertRequest {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * Mapping to attribute metered usage to the customer by the event subject.
   */
  usageAttribution?: CustomerUsageAttribution;
  /**
   * The primary email address of the customer.
   */
  primaryEmail?: string;
  /**
   * Currency of the customer. Used for billing, tax and invoicing.
   */
  currency?: string;
  /**
   * The billing address of the customer. Used for tax and invoicing.
   */
  billingAddress?: Address;
}
/**
 * Billing customer data.
 */
export interface CustomerBillingData {
  /**
   * The billing profile for the customer.
   *
   * If not provided, the default billing profile will be used.
   */
  billingProfile?: BillingProfileReference;
  /**
   * App customer data.
   */
  appData?: AppCustomerData;
}
/**
 * Billing profile reference.
 */
export interface BillingProfileReference {
  /**
   * The ID of the billing profile.
   */
  id: string;
}
/**
 * App customer data.
 */
export interface AppCustomerData {
  /**
   * Used if the customer has a linked Stripe app.
   */
  stripe?: AppCustomerDataStripe;
  /**
   * Used if the customer has a linked external invoicing app.
   */
  externalInvoicing?: AppCustomerDataExternalInvoicing;
}
/**
 * Stripe customer data.
 */
export interface AppCustomerDataStripe {
  /**
   * The Stripe customer ID used.
   */
  customerId?: string;
  /**
   * The Stripe default payment method ID.
   */
  defaultPaymentMethodId?: string;
  /**
   * Labels for this Stripe integration on the customer.
   */
  labels?: Labels;
}
/**
 * External invoicing customer data.
 */
export interface AppCustomerDataExternalInvoicing {
  /**
   * Labels for this external invoicing integration on the customer.
   */
  labels?: Labels;
}
/**
 * CustomerBillingData upsert request.
 */
export interface UpsertRequest_2 {
  /**
   * The billing profile for the customer.
   *
   * If not provided, the default billing profile will be used.
   */
  billingProfile?: BillingProfileReference;
  /**
   * App customer data.
   */
  appData?: AppCustomerData;
}
/**
 * AppCustomerData upsert request.
 */
export interface UpsertRequest_3 {
  /**
   * Used if the customer has a linked Stripe app.
   */
  stripe?: AppCustomerDataStripe;
  /**
   * Used if the customer has a linked external invoicing app.
   */
  externalInvoicing?: AppCustomerDataExternalInvoicing;
}
/**
 * Request to create a Stripe Checkout Session for the customer.
 *
 * Checkout Sessions are used to collect payment method information from customers
 * in a secure, Stripe-hosted interface. This integration uses setup mode to
 * collect payment methods that can be charged later for subscription billing.
 */
export interface CustomerBillingStripeCreateCheckoutSessionRequest {
  /**
   * Options for configuring the Stripe Checkout Session.
   *
   * These options are passed directly to Stripe's
   * [checkout session creation API](https://docs.stripe.com/api/checkout/sessions/create).
   */
  stripeOptions: CreateStripeCheckoutSessionRequestOptions;
}
/**
 * Configuration options for creating a Stripe Checkout Session.
 *
 * Based on Stripe's
 * [Checkout Session API parameters](https://docs.stripe.com/api/checkout/sessions/create).
 */
export interface CreateStripeCheckoutSessionRequestOptions {
  /**
   * Whether to collect the customer's billing address.
   *
   * Defaults to auto, which only collects the address when necessary for tax
   * calculation.
   */
  billingAddressCollection?: CreateStripeCheckoutSessionBillingAddressCollection;
  /**
   * URL to redirect customers who cancel the checkout session.
   *
   * Not allowed when ui_mode is "embedded".
   */
  cancelUrl?: string;
  /**
   * Unique reference string for reconciling sessions with internal systems.
   *
   * Can be a customer ID, cart ID, or any other identifier.
   */
  clientReferenceId?: string;
  /**
   * Controls which customer fields can be updated by the checkout session.
   */
  customerUpdate?: CreateStripeCheckoutSessionCustomerUpdate;
  /**
   * Configuration for collecting customer consent during checkout.
   */
  consentCollection?: CreateStripeCheckoutSessionConsentCollection;
  /**
   * Three-letter ISO 4217 currency code in uppercase.
   *
   * Required for payment mode sessions. Optional for setup mode sessions.
   */
  currency?: string;
  /**
   * Custom text to display during checkout at various stages.
   */
  customText?: CheckoutSessionCustomTextParams;
  /**
   * Unix timestamp when the checkout session expires.
   *
   * Can be 30 minutes to 24 hours from creation. Defaults to 24 hours.
   */
  expiresAt?: bigint;
  /**
   * IETF language tag for the checkout UI locale.
   *
   * If blank or "auto", uses the browser's locale. Example: "en", "fr", "de".
   */
  locale?: string;
  /**
   * Set of key-value pairs to attach to the checkout session.
   *
   * Useful for storing additional structured information.
   */
  metadata?: Record<string, string>;
  /**
   * Return URL for embedded checkout sessions after payment authentication.
   *
   * Required if ui_mode is "embedded" and redirect-based payment methods are
   * enabled.
   */
  returnUrl?: string;
  /**
   * Success URL to redirect customers after completing payment or setup.
   *
   * Not allowed when ui_mode is "embedded". See:
   * https://docs.stripe.com/payments/checkout/custom-success-page
   */
  successUrl?: string;
  /**
   * The UI mode for the checkout session.
   *
   * "hosted" displays a Stripe-hosted page. "embedded" integrates directly into your
   * app. Defaults to "hosted".
   */
  uiMode?: CheckoutSessionUiMode;
  /**
   * List of payment method types to enable (e.g., "card", "us_bank_account").
   *
   * If not specified, Stripe enables all relevant payment methods.
   */
  paymentMethodTypes?: Array<string>;
  /**
   * Redirect behavior for embedded checkout sessions.
   *
   * Controls when to redirect users after completion. See:
   * https://docs.stripe.com/payments/checkout/custom-success-page?payment-ui=embedded-form
   */
  redirectOnCompletion?: CreateStripeCheckoutSessionRedirectOnCompletion;
  /**
   * Configuration for collecting tax IDs during checkout.
   */
  taxIdCollection?: CreateCheckoutSessionTaxIdCollection;
}
/**
 * Controls whether Checkout collects the customer's billing address.
 */
export enum CreateStripeCheckoutSessionBillingAddressCollection {
  /**
   * Collect billing address only when necessary (e.g., for tax calculation).
   */
  Auto = "auto",
  /**
   * Always collect the customer's billing address.
   */
  Required = "required"
}
/**
 * Controls which customer fields can be updated by the checkout session.
 */
export interface CreateStripeCheckoutSessionCustomerUpdate {
  /**
   * Whether to save the billing address to customer.address.
   *
   * Defaults to "never".
   */
  address?: CreateStripeCheckoutSessionCustomerUpdateBehavior;
  /**
   * Whether to save the customer name to customer.name.
   *
   * Defaults to "never".
   */
  name?: CreateStripeCheckoutSessionCustomerUpdateBehavior;
  /**
   * Whether to save shipping information to customer.shipping.
   *
   * Defaults to "never".
   */
  shipping?: CreateStripeCheckoutSessionCustomerUpdateBehavior;
}
/**
 * Behavior for updating customer fields from checkout session.
 */
export enum CreateStripeCheckoutSessionCustomerUpdateBehavior {
  /**
   * Automatically determine whether to update the customer using session details.
   */
  Auto = "auto",
  /**
   * Never update the customer object.
   */
  Never = "never"
}
/**
 * Checkout Session consent collection configuration.
 */
export interface CreateStripeCheckoutSessionConsentCollection {
  /**
   * Controls the visibility of payment method reuse agreement.
   */
  paymentMethodReuseAgreement?: CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement;
  /**
   * Enables collection of promotional communication consent.
   *
   * Only available to US merchants. When set to "auto", Checkout determines whether
   * to show the option based on the customer's locale.
   */
  promotions?: CreateStripeCheckoutSessionConsentCollectionPromotions;
  /**
   * Requires customers to accept terms of service before payment.
   *
   * Requires a valid terms of service URL in your Stripe Dashboard settings.
   */
  termsOfService?: CreateStripeCheckoutSessionConsentCollectionTermsOfService;
}
/**
 * Payment method reuse agreement configuration.
 */
export interface CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement {
  /**
   * Position and visibility of the payment method reuse agreement.
   */
  position?: CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition;
}
/**
 * Position of payment method reuse agreement in the UI.
 */
export enum CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementPosition {
  /**
   * Use Stripe defaults for visibility and position.
   */
  Auto = "auto",
  /**
   * Hide the payment method reuse agreement.
   */
  Hidden = "hidden"
}
/**
 * Promotional communication consent collection setting.
 */
export enum CreateStripeCheckoutSessionConsentCollectionPromotions {
  /**
   * Show promotional consent option based on customer context and locale.
   */
  Auto = "auto",
  /**
   * Do not collect promotional communication consent.
   */
  None = "none"
}
/**
 * Terms of service acceptance requirement.
 */
export enum CreateStripeCheckoutSessionConsentCollectionTermsOfService {
  /**
   * Do not display terms of service checkbox.
   */
  None = "none",
  /**
   * Require customers to accept terms of service before payment.
   */
  Required = "required"
}
/**
 * Custom text displayed at various stages of the checkout flow.
 */
export interface CheckoutSessionCustomTextParams {
  /**
   * Text displayed after the payment confirmation button.
   */
  afterSubmit?: {
    /**
     * The custom message text (max 1200 characters).
     */
    message?: string;
  };
  /**
   * Text displayed alongside shipping address collection.
   */
  shippingAddress?: {
    /**
     * The custom message text (max 1200 characters).
     */
    message?: string;
  };
  /**
   * Text displayed alongside the payment confirmation button.
   */
  submit?: {
    /**
     * The custom message text (max 1200 characters).
     */
    message?: string;
  };
  /**
   * Text replacing the default terms of service agreement text.
   */
  termsOfServiceAcceptance?: {
    /**
     * The custom message text (max 1200 characters).
     */
    message?: string;
  };
}
/**
 * A 64-bit integer. (`-9,223,372,036,854,775,808` to `9,223,372,036,854,775,807`)
 */
export type Int64 = bigint;
/**
 * Checkout Session UI mode.
 */
export enum CheckoutSessionUiMode {
  /**
   * Checkout UI embedded directly in your application.
   */
  Embedded = "embedded",
  /**
   * Checkout UI hosted on a Stripe-provided page.
   */
  Hosted = "hosted"
}
/**
 * Redirect behavior for embedded checkout sessions.
 */
export enum CreateStripeCheckoutSessionRedirectOnCompletion {
  /**
   * Always redirect to return_url after successful confirmation.
   */
  Always = "always",
  /**
   * Redirect only when a redirect-based payment method is used.
   */
  IfRequired = "if_required",
  /**
   * Never redirect, and disable redirect-based payment methods.
   */
  Never = "never"
}
/**
 * Tax ID collection configuration for checkout sessions.
 */
export interface CreateCheckoutSessionTaxIdCollection {
  /**
   * Enable tax ID collection during checkout.
   *
   * Defaults to false.
   */
  enabled?: boolean;
  /**
   * Whether tax ID collection is required.
   *
   * Defaults to "never".
   */
  required?: CreateCheckoutSessionTaxIdCollectionRequired;
}
/**
 * Tax ID collection requirement level.
 */
export enum CreateCheckoutSessionTaxIdCollectionRequired {
  /**
   * Require tax ID if collection is supported for the billing address country.
   *
   * See: https://docs.stripe.com/tax/checkout/tax-ids#supported-types
   */
  IfSupported = "if_supported",
  /**
   * Tax ID collection is never required.
   */
  Never = "never"
}
/**
 * Result of creating a Stripe Checkout Session.
 *
 * Contains all the information needed to redirect customers to the checkout or
 * initialize an embedded checkout flow.
 */
export interface CreateStripeCheckoutSessionResult {
  /**
   * The customer ID in the billing system.
   */
  customerId: string;
  /**
   * The Stripe customer ID.
   */
  stripeCustomerId: string;
  /**
   * The Stripe checkout session ID.
   */
  sessionId: string;
  /**
   * The setup intent ID created for collecting the payment method.
   */
  setupIntentId: string;
  /**
   * Client secret for initializing Stripe.js on the client side.
   *
   * Required for embedded checkout sessions. See:
   * https://docs.stripe.com/payments/checkout/custom-success-page
   */
  clientSecret?: string;
  /**
   * The client reference ID provided in the request.
   *
   * Useful for reconciling the session with your internal systems.
   */
  clientReferenceId?: string;
  /**
   * Customer's email address if provided to Stripe.
   */
  customerEmail?: string;
  /**
   * Currency code for the checkout session.
   */
  currency?: string;
  /**
   * Timestamp when the checkout session was created.
   */
  createdAt: Date;
  /**
   * Timestamp when the checkout session will expire.
   */
  expiresAt?: Date;
  /**
   * Metadata attached to the checkout session.
   */
  metadata?: Record<string, string>;
  /**
   * The status of the checkout session.
   *
   * See:
   * https://docs.stripe.com/api/checkout/sessions/object#checkout_session_object-status
   */
  status?: string;
  /**
   * URL to redirect customers to the checkout page (for hosted mode).
   */
  url?: string;
  /**
   * Mode of the checkout session.
   *
   * Currently only "setup" mode is supported for collecting payment methods.
   */
  mode: StripeCheckoutSessionMode;
  /**
   * The cancel URL where customers are redirected if they cancel.
   */
  cancelUrl?: string;
  /**
   * The success URL where customers are redirected after completion.
   */
  successUrl?: string;
  /**
   * The return URL for embedded sessions after authentication.
   */
  returnUrl?: string;
}
/**
 * Stripe Checkout Session mode.
 *
 * Determines the primary purpose of the checkout session.
 */
export enum StripeCheckoutSessionMode {
  /**
   * Collect payment method information for later use.
   *
   * Used for subscription billing where the payment method is charged later.
   */
  Setup = "setup"
}
/**
 * Request to create a Stripe Customer Portal Session for the customer.
 *
 * Useful to redirect the customer to the Stripe Customer Portal to manage their
 * payment methods, change their billing address and access their invoice history.
 * Only returns URL if the customer billing profile is linked to a stripe app and
 * customer.
 */
export interface CustomerBillingStripeCreateCustomerPortalSessionRequest {
  /**
   * Options for configuring the Stripe Customer Portal Session.
   */
  stripeOptions: CreateStripeCustomerPortalSessionOptions;
}
/**
 * Request to create a Stripe Customer Portal Session.
 */
export interface CreateStripeCustomerPortalSessionOptions {
  /**
   * The ID of an existing
   * [Stripe configuration](https://docs.stripe.com/api/customer_portal/configurations)
   * to use for this session, describing its functionality and features. If not
   * specified, the session uses the default configuration.
   */
  configurationId?: string;
  /**
   * The IETF
   * [language tag](https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-locale)
   * of the locale customer portal is displayed in. If blank or `auto`, the
   * customer's preferred_locales or browser's locale is used.
   */
  locale?: string;
  /**
   * The
   * [URL to redirect](https://docs.stripe.com/api/customer_portal/sessions/create#create_portal_session-return_url)
   * the customer to after they have completed their requested actions.
   */
  returnUrl?: string;
}
/**
 * Result of creating a
 * [Stripe Customer Portal Session](https://docs.stripe.com/api/customer_portal/sessions/object).
 *
 * Contains all the information needed to redirect the customer to the Stripe
 * Customer Portal.
 */
export interface CreateStripeCustomerPortalSessionResult {
  /**
   * The ID of the customer portal session.
   *
   * See:
   * https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-id
   */
  id: string;
  /**
   * The ID of the stripe customer.
   */
  stripeCustomerId: string;
  /**
   * Configuration used to customize the customer portal.
   *
   * See:
   * https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-configuration
   */
  configurationId: string;
  /**
   * Livemode.
   *
   * See:
   * https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-livemode
   */
  livemode: boolean;
  /**
   * Created at.
   *
   * See:
   * https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-created
   */
  createdAt: Date;
  /**
   * Return URL.
   *
   * See:
   * https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-return_url
   */
  returnUrl: string;
  /**
   * The IETF language tag of the locale customer portal is displayed in.
   *
   * See:
   * https://docs.stripe.com/api/customer_portal/sessions/object#portal_session_object-locale
   */
  locale: string;
  /**
   * The URL to redirect the customer to after they have completed their requested
   * actions.
   */
  url: string;
}
/**
 * List customer entitlement access response data.
 */
export interface ListCustomerEntitlementAccessResponseData {
  /**
   * The list of entitlement access results.
   */
  data: Array<EntitlementAccessResult>;
}
/**
 * Entitlement access result.
 */
export interface EntitlementAccessResult {
  /**
   * The type of the entitlement.
   */
  type: EntitlementType;
  /**
   * The feature key of the entitlement.
   */
  featureKey: string;
  /**
   * Whether the customer has access to the feature. Always true for `boolean` and
   * `static` entitlements. Depends on balance for `metered` entitlements.
   */
  hasAccess: boolean;
  /**
   * Only available for static entitlements. Config is the JSON parsable
   * configuration of the entitlement. Useful to describe per customer configuration.
   */
  config?: string;
}
/**
 * The type of the entitlement.
 */
export enum EntitlementType {
  Metered = "metered",
  Static = "static",
  Boolean = "boolean"
}
/**
 * CreditGrant create request.
 */
export interface CreateRequestNested {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * Funding method of the grant.
   */
  fundingMethod: CreditFundingMethod;
  /**
   * The currency of the granted credits.
   */
  currency: CurrencyCode_2;
  /**
   * Granted credit amount.
   */
  amount: string;
  /**
   * Purchase and payment terms of the grant. Present when a funding workflow applies
   * (funding_method is not `none`).
   */
  purchase?: {
    /**
     * Currency of the purchase amount.
     */
    currency: string;
    /**
     * Cost basis per credit unit used to calculate the purchase amount.
     *
     * If `per_unit_cost_basis` is 0.50 and credit amount is $100.00, the total charge
     * is $50.00. The value must be greater than 0. If the cost basis is 0, use
     * `funding_method=none` instead.
     *
     * Defaults to 1.0.
     */
    perUnitCostBasis?: string;
    /**
     * Controls when credits become available for consumption.
     *
     * Defaults to `on_creation`.
     */
    availabilityPolicy?: CreditAvailabilityPolicy;
  };
  /**
   * Tax configuration for the grant.
   *
   * For `invoice` and `external` funding methods, tax configuration should be
   * provided to ensure correct revenue recognition. When not provided, the default
   * credit grant tax code is applied, if that's not set the global default taxcode
   * is used.
   */
  taxConfig?: CreditGrantTaxConfig;
  /**
   * Filters for the credit grant.
   */
  filters?: {
    /**
     * Limit the credit grant to specific features. If no features are specified, the
     * credit grant can be used for any feature.
     */
    features?: Array<string>;
  };
  /**
   * Draw-down priority of the grant. Lower values have higher priority.
   */
  priority?: number;
}
/**
 * The funding method describes how the grant is funded.
 *
 * - `none`: No funding workflow applies, for example promotional grants
 * - `invoice`: The grant is funded by an in-system invoice flow
 * - `external`: The grant is funded outside the system (e.g., wire transfer,
 * external invoice, or manual reconciliation)
 */
export enum CreditFundingMethod {
  None = "none",
  Invoice = "invoice",
  External = "external"
}
/**
 * Fiat or custom currency code.
 */
export type CurrencyCode_2 = string;
/**
 * When credits become available for consumption.
 *
 * - `on_creation`: Credits are available as soon as the grant is created.
 * - `on_authorization`: Credits are available once the payment is authorized.
 * - `on_settlement`: Credits are available once the payment is settled.
 */
export enum CreditAvailabilityPolicy {
  OnCreation = "on_creation"
}
/**
 * Tax configuration for a credit grant.
 *
 * Tax configuration should be provided to ensure correct revenue recognition,
 * including for externally funded grants.
 */
export interface CreditGrantTaxConfig {
  /**
   * Tax behavior applied to the invoice line item.
   */
  behavior?: TaxBehavior;
  /**
   * Tax code applied to the invoice line item.
   */
  taxCode?: TaxCodeReference;
}
/**
 * Tax behavior.
 *
 * This enum is used to specify whether tax is included in the price or excluded
 * from the price.
 */
export enum TaxBehavior {
  /**
   * Tax is included in the price.
   */
  Inclusive = "inclusive",
  /**
   * Tax is excluded from the price.
   */
  Exclusive = "exclusive"
}
/**
 * Reference to a tax code.
 */
export interface TaxCodeReference extends ResourceReference_2 {}
/**
 * TaxCode reference.
 */
export interface ResourceReference_2 {
  id: string;
}
/**
 * A 16-bit integer. (`-32,768` to `32,767`)
 */
export type Int16 = number;
/**
 * A 32-bit integer. (`-2,147,483,648` to `2,147,483,647`)
 */
export type Int32 = number;
/**
 * A credit grant allocates credits to a customer.
 *
 * Credits are drawn down against charges according to the settlement mode
 * configured on the rate card.
 */
export interface CreditGrant {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  /**
   * Funding method of the grant.
   */
  fundingMethod: CreditFundingMethod;
  /**
   * The currency of the granted credits.
   */
  currency: CurrencyCode_2;
  /**
   * Granted credit amount.
   */
  amount: string;
  /**
   * Purchase and payment terms of the grant. Present when a funding workflow applies
   * (funding_method is not `none`).
   */
  purchase?: {
    /**
     * Currency of the purchase amount.
     */
    currency: string;
    /**
     * Cost basis per credit unit used to calculate the purchase amount.
     *
     * If `per_unit_cost_basis` is 0.50 and credit amount is $100.00, the total charge
     * is $50.00. The value must be greater than 0. If the cost basis is 0, use
     * `funding_method=none` instead.
     *
     * Defaults to 1.0.
     */
    perUnitCostBasis?: string;
    /**
     * The purchase amount. Calculated from `per_unit_cost_basis` and credit `amount`.
     */
    amount: string;
    /**
     * Controls when credits become available for consumption.
     *
     * Defaults to `on_creation`.
     */
    availabilityPolicy?: CreditAvailabilityPolicy;
    /**
     * Current payment settlement status.
     */
    settlementStatus?: CreditPurchasePaymentSettlementStatus;
  };
  /**
   * Tax configuration for the grant.
   *
   * For `invoice` and `external` funding methods, tax configuration should be
   * provided to ensure correct revenue recognition. When not provided, the default
   * credit grant tax code is applied, if that's not set the global default taxcode
   * is used.
   */
  taxConfig?: CreditGrantTaxConfig;
  /**
   * Invoice references for the grant. Available when `funding_method` is `invoice`.
   */
  invoice?: {
    /**
     * Identifier of the invoice associated with the grant.
     */
    id?: string;
    /**
     * Identifier of the invoice line associated with the grant.
     */
    line?: {
      id: string;
    };
  };
  /**
   * Filters for the credit grant.
   */
  filters?: {
    /**
     * Limit the credit grant to specific features. If no features are specified, the
     * credit grant can be used for any feature.
     */
    features?: Array<string>;
  };
  /**
   * Draw-down priority of the grant. Lower values have higher priority.
   */
  priority?: number;
  /**
   * Timestamp when the grant was voided.
   */
  voidedAt?: Date;
  /**
   * Current lifecycle status of the grant.
   */
  status: CreditGrantStatus;
}
/**
 * Credit purchase payment settlement status.
 *
 * - `pending`: Payment has been initiated and is not yet authorized.
 * - `authorized`: Payment has been authorized.
 * - `settled`: Payment has been settled.
 */
export enum CreditPurchasePaymentSettlementStatus {
  Pending = "pending",
  Authorized = "authorized",
  Settled = "settled"
}
/**
 * Credit grant lifecycle status.
 *
 * - `pending`: The credit block has been created but is not yet valid.
 * (`effective_at` is in the future or availability_policy is not met)
 * - `active`: The credit block is currently valid and eligible for consumption.
 * (`effective_at` is in the past, `expires_at` is in the future and
 * availability_policy is met)
 * - `expired`: The credit block expired with remaining unused balance,
 * `expires_at` time has passed.
 * - `voided`: The credit block was voided. Remaining balance is forfeited.
 */
export enum CreditGrantStatus {
  Pending = "pending",
  Active = "active",
  Expired = "expired",
  Voided = "voided"
}
/**
 * Filter options for listing credit grants.
 */
export interface ListCreditGrantsParamsFilter {
  /**
   * Filter credit grants by status.
   */
  status?: CreditGrantStatus;
  /**
   * Filter credit grants by currency.
   */
  currency?: string;
}
/**
 * Filter options for getting a credit balance.
 */
export interface GetCreditBalanceParamsFilter {
  /**
   * Filter credit balance by currency.
   */
  currency?: CurrencyCode_2;
}
/**
 * The balances of the credits of a customer.
 */
export interface CreditBalances {
  /**
   * The timestamp of the balance retrieval.
   */
  retrievedAt: Date;
  /**
   * The balances by currencies.
   */
  balances: Array<CreditBalance>;
}
/**
 * The credit balance by currency.
 */
export interface CreditBalance {
  currency: CurrencyCode_2;
  /**
   * Credits that have been granted but cannot yet be consumed. Includes grants
   * awaiting payment clearance or with a future effective date.
   */
  pending: string;
  /**
   * Credits that can be consumed right now. Derived from cleared grants after
   * applying eligibility and restriction rules.
   */
  available: string;
}
/**
 * CreditAdjustment create request.
 */
export interface CreateRequest_3 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * The currency of the granted credits.
   */
  currency: CurrencyCode_2;
  /**
   * Granted credit amount.
   */
  amount: string;
}
/**
 * A credit adjustment can be used to make manual adjustments to a customer's
 * credit balance.
 *
 * Supported use-cases:
 *
 * - Usage correction
 */
export interface CreditAdjustment {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * The currency of the granted credits.
   */
  currency: CurrencyCode_2;
  /**
   * Granted credit amount.
   */
  amount: string;
}
/**
 * Request body for updating the external payment settlement status of a credit
 * grant.
 */
export interface UpdateCreditGrantExternalSettlementRequest {
  /**
   * The new payment settlement status.
   */
  status: CreditPurchasePaymentSettlementStatus;
}
/**
 * Filter options for listing credit transactions.
 */
export interface ListCreditTransactionsParamsFilter {
  /**
   * Filter credit transactions by type.
   */
  type?: CreditTransactionType;
  /**
   * Filter credit transactions by currency.
   */
  currency?: CurrencyCode_2;
}
/**
 * The type of the credit transaction.
 *
 * - `funded`: Credit granted and available for consumption.
 * - `consumed`: Credit consumed by usage or fees.
 */
export enum CreditTransactionType {
  Funded = "funded",
  Consumed = "consumed"
}
/**
 * A credit transaction represents a single credit movement on the customer's
 * balance.
 *
 * Credit transactions are immutable.
 */
export interface CreditTransaction {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * The date and time the transaction was booked.
   */
  bookedAt: Date;
  /**
   * The type of credit transaction.
   */
  type: CreditTransactionType;
  /**
   * Currency of the balance affected by the transaction.
   */
  currency: CurrencyCode_2;
  /**
   * Signed amount of the credit movement. Positive values add balance, negative
   * values reduce balance.
   */
  amount: string;
  /**
   * The available balance before and after the transaction.
   */
  availableBalance: {
    before: string;
    after: string;
  };
}
/**
 * Filter options for listing charges.
 */
export interface ListCustomerChargesParamsFilter {
  /**
   * Filter charges by status.
   *
   * Supported statuses are:
   *
   * - `created`
   * - `active`
   * - `final`
   * - `deleted`
   *
   * If omitted, all statuses are returned except for `deleted`.
   */
  status?: StringFieldFilterExact;
}
/**
 * Filters on the given string field value by exact match. All properties are
 * optional; provide exactly one to specify the comparison.
 */
export type StringFieldFilterExact = string | {
  /**
   * Value strictly equals the given string value.
   */
  eq?: string;
  /**
   * Returns entities that exact match any of the comma-delimited phrases in the
   * filter string.
   */
  oeq?: Array<string>;
  /**
   * Value does not equal the given string value.
   */
  neq?: string;
};
/**
 * Expands for customer charges.
 *
 * Values:
 *
 * - `real_time_usage`: The charge's real-time usage.
 */
export enum ChargesExpand {
  RealTimeUsage = "real_time_usage"
}
/**
 * Customer charge.
 */
export type Charge = {
  type: "flat_fee"
} & FlatFeeCharge | {
  type: "usage_based"
} & UsageBasedCharge;
/**
 * A flat fee charge for a customer.
 */
export interface FlatFeeCharge {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  /**
   * The type of the charge.
   */
  type: ChargeType.FlatFee;
  /**
   * The customer owning the charge.
   */
  customer: CustomerReference;
  /**
   * The charge is managed by the following entity.
   */
  managedBy: ResourceManagedBy;
  /**
   * The subscription that originated the charge, when the charge was created from a
   * subscription item.
   */
  subscription?: SubscriptionReference;
  /**
   * The currency of the charge.
   */
  currency: string;
  /**
   * The lifecycle status of the charge.
   */
  status: ChargeStatus;
  /**
   * The timestamp when the charge is intended to be invoiced.
   */
  invoiceAt: Date;
  /**
   * The effective service period covered by the charge.
   */
  servicePeriod: ClosedPeriod;
  /**
   * The full, unprorated service period of the charge.
   */
  fullServicePeriod: ClosedPeriod;
  /**
   * The billing period the charge belongs to.
   */
  billingPeriod: ClosedPeriod;
  /**
   * The earliest time when the charge should be advanced again by background
   * processing.
   */
  advanceAfter?: Date;
  /**
   * The price of the charge.
   */
  price: Price;
  /**
   * Unique reference ID of the charge.
   */
  uniqueReferenceId?: string;
  /**
   * Settlement mode of the charge.
   */
  settlementMode: SettlementMode;
  /**
   * Tax configuration of the charge.
   */
  taxConfig?: TaxConfig;
  /**
   * Payment term of the flat fee charge.
   */
  paymentTerm: PricePaymentTerm;
  /**
   * The discounts applied to the charge.
   */
  discounts?: FlatFeeDiscounts;
  /**
   * The feature associated with the charge, when applicable.
   */
  featureKey?: string;
  /**
   * The proration configuration of the charge.
   */
  prorationConfiguration: ProrationConfiguration;
  /**
   * The amount after proration of the charge.
   */
  amountAfterProration: CurrencyAmount;
}
/**
 * Type of a charge.
 *
 * Values:
 *
 * - `flat_fee`: A fixed-amount charge.
 * - `usage_based`: A usage-priced charge.
 */
export enum ChargeType {
  FlatFee = "flat_fee",
  UsageBased = "usage_based"
}
/**
 * Customer reference.
 */
export interface CustomerReference {
  /**
   * The ID of the customer.
   */
  id: string;
}
/**
 * Identifies which system manages a resource.
 *
 * Values:
 *
 * - `manual`: The resource is managed manually (overridden by our API users).
 * - `system`: The resource is managed by the system.
 * - `subscription`: The resource is managed by the subscription.
 */
export enum ResourceManagedBy {
  Manual = "manual",
  System = "system",
  Subscription = "subscription"
}
/**
 * Subscription reference represents a reference to the specific subscription item
 * this entity represents.
 */
export interface SubscriptionReference {
  /**
   * The ID of the subscription.
   */
  id: string;
  /**
   * The phase of the subscription.
   */
  phase: {
    /**
     * The ID of the phase.
     */
    id: string;
    /**
     * The item of the phase.
     */
    item: {
      /**
       * The ID of the item.
       */
      id: string;
    };
  };
}
/**
 * Lifecycle status of a charge.
 *
 * Values:
 *
 * - `created`: The charge has been created but is not active yet.
 * - `active`: The charge is active.
 * - `final`: The charge is fully finalized and no further changes are expected.
 * - `deleted`: The charge has been deleted.
 */
export enum ChargeStatus {
  Created = "created",
  Active = "active",
  Final = "final",
  Deleted = "deleted"
}
/**
 * A period with defined start and end dates.
 *
 * The period is always inclusive at the start and exclusive at the end.
 */
export interface ClosedPeriod {
  /**
   * The start of the period.
   *
   * The period is inclusive at the start.
   */
  from: Date;
  /**
   * The end of the period.
   *
   * The period is exclusive at the end.
   */
  to: Date;
}
/**
 * Price.
 */
export type Price = {
  type: "free"
} & PriceFree | {
  type: "flat"
} & PriceFlat | {
  type: "unit"
} & PriceUnit | {
  type: "graduated"
} & PriceGraduated | {
  type: "volume"
} & PriceVolume;
/**
 * Free price.
 */
export interface PriceFree {
  /**
   * The type of the price.
   */
  type: PriceType.Free;
}
/**
 * The type of the price.
 *
 * - `free`: No charge, the rate card is included at no cost.
 * - `flat`: A fixed amount charged once per billing period, regardless of usage.
 * - `unit`: A fixed rate charged per billing unit consumed.
 * - `graduated`: Tiered pricing where each tier's rate applies only to usage
 * within that tier.
 * - `volume`: Tiered pricing where the rate for the highest tier reached applies
 * to all units in the period.
 */
export enum PriceType {
  Free = "free",
  Flat = "flat",
  Unit = "unit",
  Graduated = "graduated",
  Volume = "volume"
}
/**
 * Flat price.
 */
export interface PriceFlat {
  /**
   * The type of the price.
   */
  type: PriceType.Flat;
  /**
   * The amount of the flat price.
   */
  amount: string;
}
/**
 * Unit price.
 *
 * Charges a fixed rate per billing unit. When UnitConfig is present on the rate
 * card, billing units are the converted quantities (e.g. GB instead of bytes).
 */
export interface PriceUnit {
  /**
   * The type of the price.
   */
  type: PriceType.Unit;
  /**
   * The amount of the unit price.
   */
  amount: string;
}
/**
 * Graduated tiered price.
 *
 * Each tier's rate applies only to the usage within that tier. Pricing can change
 * as cumulative usage crosses tier boundaries.
 *
 * When UnitConfig is present on the rate card, tier boundaries (up_to_amount) are
 * expressed in converted billing units.
 */
export interface PriceGraduated {
  /**
   * The type of the price.
   */
  type: PriceType.Graduated;
  /**
   * The tiers of the graduated price. At least one tier is required.
   */
  tiers: Array<PriceTier>;
}
/**
 * A price tier used in graduated and volume pricing.
 *
 * At least one price component (flat_price or unit_price) must be set. When
 * UnitConfig is present on the rate card, up_to_amount is expressed in converted
 * billing units.
 */
export interface PriceTier {
  /**
   * Up to and including this quantity will be contained in the tier. If undefined,
   * the tier is open-ended (the last tier).
   */
  upToAmount?: string;
  /**
   * The flat price component of the tier. Charged once when the tier is entered.
   */
  flatPrice?: PriceFlat;
  /**
   * The unit price component of the tier. Charged per billing unit within the tier.
   */
  unitPrice?: PriceUnit;
}
/**
 * Volume tiered price.
 *
 * The maximum quantity within a period determines the per-unit price for all units
 * in that period.
 *
 * When UnitConfig is present on the rate card, tier boundaries (up_to_amount) are
 * expressed in converted billing units.
 */
export interface PriceVolume {
  /**
   * The type of the price.
   */
  type: PriceType.Volume;
  /**
   * The tiers of the volume price. At least one tier is required.
   */
  tiers: Array<PriceTier>;
}
/**
 * Settlement mode for billing.
 *
 * Values:
 *
 * - `credit_then_invoice`: Credits are applied first, then any remainder is
 * invoiced.
 * - `credit_only`: Usage is settled exclusively against credits.
 */
export enum SettlementMode {
  CreditThenInvoice = "credit_then_invoice",
  CreditOnly = "credit_only"
}
/**
 * Set of provider specific tax configs.
 */
export interface TaxConfig {
  /**
   * Tax behavior.
   *
   * If not specified the billing profile is used to determine the tax behavior. If
   * not specified in the billing profile, the provider's default behavior is used.
   */
  behavior?: TaxBehavior;
  /**
   * Stripe tax config.
   */
  stripe?: TaxConfigStripe;
  /**
   * External invoicing tax config.
   */
  externalInvoicing?: TaxConfigExternalInvoicing;
  /**
   * Tax code ID.
   */
  taxCodeId?: string;
  /**
   * Tax code reference.
   *
   * When both `tax_code` and `tax_code_id` are provided, `tax_code` takes
   * precedence. When `stripe.code` is also provided, `tax_code` still wins and
   * `stripe.code` is ignored.
   */
  taxCode?: TaxCodeReference;
}
/**
 * The tax config for Stripe.
 */
export interface TaxConfigStripe {
  /**
   * Product [tax code](https://docs.stripe.com/tax/tax-codes).
   */
  code: string;
}
/**
 * External invoicing tax config.
 */
export interface TaxConfigExternalInvoicing {
  /**
   * The tax code should be interpreted by the external invoicing provider.
   */
  code: string;
}
/**
 * The payment term of a flat price.
 */
export type PricePaymentTerm = "in_advance" | "in_arrears";
/**
 * Discounts applicable to flat fee charges.
 *
 * This is the same as `ProductCatalog.Discounts` but without the `usage` field,
 * which is not applicable to flat fee charges.
 */
export interface FlatFeeDiscounts {
  /**
   * Percentage discount applied to the price (0–100).
   */
  percentage?: number;
}
/**
 * A number with decimal value
 */
export type Float = number;
/**
 * The proration configuration of the rate card.
 */
export interface ProrationConfiguration {
  /**
   * The proration mode of the rate card.
   */
  mode: ProrationMode;
}
/**
 * The proration mode of the rate card.
 *
 * Values:
 *
 * - `no_proration`: No proration.
 * - `prorate_prices`: Prorate the price based on the time remaining in the billing
 * period.
 */
export enum ProrationMode {
  NoProration = "no_proration",
  ProratePrices = "prorate_prices"
}
/**
 * Monetary amount in a specific currency.
 */
export interface CurrencyAmount {
  amount: string;
  currency: string;
}
/**
 * A usage-based charge for a customer.
 */
export interface UsageBasedCharge {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  /**
   * The type of the charge.
   */
  type: ChargeType.UsageBased;
  /**
   * The customer owning the charge.
   */
  customer: CustomerReference;
  /**
   * The charge is managed by the following entity.
   */
  managedBy: ResourceManagedBy;
  /**
   * The subscription that originated the charge, when the charge was created from a
   * subscription item.
   */
  subscription?: SubscriptionReference;
  /**
   * The currency of the charge.
   */
  currency: string;
  /**
   * The lifecycle status of the charge.
   */
  status: ChargeStatus;
  /**
   * The timestamp when the charge is intended to be invoiced.
   */
  invoiceAt: Date;
  /**
   * The effective service period covered by the charge.
   */
  servicePeriod: ClosedPeriod;
  /**
   * The full, unprorated service period of the charge.
   */
  fullServicePeriod: ClosedPeriod;
  /**
   * The billing period the charge belongs to.
   */
  billingPeriod: ClosedPeriod;
  /**
   * The earliest time when the charge should be advanced again by background
   * processing.
   */
  advanceAfter?: Date;
  /**
   * The price of the charge.
   */
  price: Price;
  /**
   * Unique reference ID of the charge.
   */
  uniqueReferenceId?: string;
  /**
   * Settlement mode of the charge.
   */
  settlementMode: SettlementMode;
  /**
   * Tax configuration of the charge.
   */
  taxConfig?: TaxConfig;
  /**
   * Discounts applied to the usage-based charge.
   */
  discounts?: Discounts;
  /**
   * The feature associated with the charge.
   */
  featureKey: string;
  /**
   * Aggregated booked and realtime totals for the charge.
   */
  totals: ChargeTotals;
}
/**
 * Discount configuration for a rate card.
 */
export interface Discounts {
  /**
   * Percentage discount applied to the price (0–100).
   */
  percentage?: number;
  /**
   * Number of usage units granted free before billing starts. Only applies to
   * usage-based lines (not flat fees). Usage is treated as zero until this amount is
   * exhausted.
   */
  usage?: string;
}
/**
 * The totals of a change.
 *
 * RealTime is only expanded when the `real_time_usage` expand is used.
 */
export interface ChargeTotals {
  /**
   * The amount of the charge already booked to the internal accounting system.
   */
  booked: BillingTotals;
  /**
   * The realtime amount of the charge.
   *
   * Requires the `realtime_usage` expand.
   */
  realtime?: BillingTotals;
}
/**
 * Totals contains the summaries of all calculations for a billing resource.
 */
export interface BillingTotals {
  /**
   * The total value of the resource before taxes, discounts and commitments.
   */
  amount: string;
  /**
   * The total tax amount applied to the resource.
   */
  taxesTotal: string;
  /**
   * The total tax amount already included in the resource amount.
   */
  taxesInclusiveTotal: string;
  /**
   * The total tax amount added on top of the resource amount.
   */
  taxesExclusiveTotal: string;
  /**
   * The total amount contributed by additional charges.
   */
  chargesTotal: string;
  /**
   * The total amount deducted through discounts.
   */
  discountsTotal: string;
  /**
   * The total amount deducted through credits before taxes are applied.
   */
  creditsTotal: string;
  /**
   * The final total value of the resource after taxes, discounts and commitments.
   */
  total: string;
}
/**
 * Subscription create request.
 */
export interface SubscriptionCreate {
  labels?: Labels;
  /**
   * The customer to create the subscription for.
   */
  customer: {
    /**
     * The ID of the customer to create the subscription for.
     *
     * Either customer ID or customer key must be provided. If both are provided, the
     * ID will be used.
     */
    id?: string;
    /**
     * The key of the customer to create the subscription for.
     *
     * Either customer ID or customer key must be provided. If both are provided, the
     * ID will be used.
     */
    key?: string;
  };
  /**
   * The plan reference of the subscription.
   */
  plan: {
    /**
     * The plan ID of the subscription. Set if subscription is created from a plan.
     *
     * ID or Key of the plan is required if creating a subscription from a plan. If
     * both are provided, the ID will be used.
     */
    id?: string;
    /**
     * The plan Key of the subscription, if any. Set if subscription is created from a
     * plan.
     *
     * ID or Key of the plan is required if creating a subscription from a plan. If
     * both are provided, the ID will be used.
     */
    key?: string;
    /**
     * The plan version of the subscription, if any. If not provided, the latest
     * version of the plan will be used.
     */
    version?: number;
  };
  /**
   * A billing anchor is the fixed point in time that determines the subscription's
   * recurring billing cycle. It affects when charges occur and how prorations are
   * calculated. Common anchors:
   *
   * - Calendar month (1st of each month): `2025-01-01T00:00:00Z`
   * - Subscription anniversary (day customer signed up)
   * - Custom date (customer-specified day)
   *
   * If not provided, the subscription will be created with the subscription's
   * creation time as the billing anchor.
   */
  billingAnchor?: Date;
}
/**
 * Subscription.
 */
export interface Subscription {
  id: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  /**
   * The customer ID of the subscription.
   */
  customerId: string;
  /**
   * The plan ID of the subscription. Set if subscription is created from a plan.
   */
  planId?: string;
  /**
   * A billing anchor is the fixed point in time that determines the subscription's
   * recurring billing cycle. It affects when charges occur and how prorations are
   * calculated. Common anchors:
   *
   * - Calendar month (1st of each month): `2025-01-01T00:00:00Z`
   * - Subscription anniversary (day customer signed up)
   * - Custom date (customer-specified day)
   */
  billingAnchor: Date;
  /**
   * The status of the subscription.
   */
  status: SubscriptionStatus;
}
/**
 * Subscription status.
 */
export enum SubscriptionStatus {
  Active = "active",
  Inactive = "inactive",
  Canceled = "canceled",
  Scheduled = "scheduled"
}
/**
 * Filter options for listing subscriptions.
 */
export interface ListSubscriptionsParamsFilter {
  id?: UlidFieldFilter;
  customerId?: UlidFieldFilter;
  status?: StringFieldFilterExact;
  planId?: UlidFieldFilter;
  planKey?: StringFieldFilterExact;
}
/**
 * Request for canceling a subscription.
 */
export interface SubscriptionCancel {
  /**
   * If not provided the subscription is canceled immediately.
   */
  timing?: SubscriptionEditTiming;
}
/**
 * Subscription edit timing defined when the changes should take effect. If the
 * provided configuration is not supported by the subscription, an error will be
 * returned.
 */
export type SubscriptionEditTiming = SubscriptionEditTimingEnum | Date;
/**
 * Subscription edit timing. When immediate, the requested changes take effect
 * immediately. When next_billing_cycle, the requested changes take effect at the
 * next billing cycle.
 */
export enum SubscriptionEditTimingEnum {
  Immediate = "immediate",
  NextBillingCycle = "next_billing_cycle"
}
/**
 * Request for changing a subscription.
 */
export interface SubscriptionChange {
  labels?: Labels;
  /**
   * The customer to create the subscription for.
   */
  customer: {
    /**
     * The ID of the customer to create the subscription for.
     *
     * Either customer ID or customer key must be provided. If both are provided, the
     * ID will be used.
     */
    id?: string;
    /**
     * The key of the customer to create the subscription for.
     *
     * Either customer ID or customer key must be provided. If both are provided, the
     * ID will be used.
     */
    key?: string;
  };
  /**
   * The plan reference of the subscription.
   */
  plan: {
    /**
     * The plan ID of the subscription. Set if subscription is created from a plan.
     *
     * ID or Key of the plan is required if creating a subscription from a plan. If
     * both are provided, the ID will be used.
     */
    id?: string;
    /**
     * The plan Key of the subscription, if any. Set if subscription is created from a
     * plan.
     *
     * ID or Key of the plan is required if creating a subscription from a plan. If
     * both are provided, the ID will be used.
     */
    key?: string;
    /**
     * The plan version of the subscription, if any. If not provided, the latest
     * version of the plan will be used.
     */
    version?: number;
  };
  /**
   * A billing anchor is the fixed point in time that determines the subscription's
   * recurring billing cycle. It affects when charges occur and how prorations are
   * calculated. Common anchors:
   *
   * - Calendar month (1st of each month): `2025-01-01T00:00:00Z`
   * - Subscription anniversary (day customer signed up)
   * - Custom date (customer-specified day)
   *
   * If not provided, the subscription will be created with the subscription's
   * creation time as the billing anchor.
   */
  billingAnchor?: Date;
  /**
   * Timing configuration for the change, when the change should take effect. For
   * changing a subscription, the accepted values depend on the subscription
   * configuration.
   */
  timing: SubscriptionEditTiming;
}
/**
 * Response for changing a subscription.
 */
export interface SubscriptionChangeResponse {
  /**
   * The current subscription before the change.
   */
  current: Subscription;
  /**
   * The new state of the subscription after the change.
   */
  next: Subscription;
}
/**
 * Addon purchased with a subscription.
 */
export interface SubscriptionAddon {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  /**
   * The add-on associated with the subscription.
   */
  addon: ResourceReference_3;
  /**
   * The quantity of the add-on. Always 1 for single instance add-ons.
   */
  quantity: number;
  /**
   * An ISO-8601 timestamp representation of which point in time the quantity was
   * resolved to.
   */
  quantityAt: Date;
  /**
   * An ISO-8601 timestamp representation of the cadence start of the resource.
   */
  activeFrom: Date;
  /**
   * An ISO-8601 timestamp representation of the cadence end of the resource.
   */
  activeTo?: Date;
}
/**
 * Addon reference.
 */
export interface ResourceReference_3 {
  id: string;
}
/**
 * Installed application.
 */
export type App = {
  type: "stripe"
} & AppStripe | {
  type: "sandbox"
} & AppSandbox | {
  type: "external_invoicing"
} & AppExternalInvoicing;
/**
 * Stripe app.
 */
export interface AppStripe {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  /**
   * The app type.
   */
  type: AppType.Stripe;
  /**
   * The app catalog definition that this installed app is based on.
   */
  definition: AppCatalogItem;
  /**
   * Status of the app connection.
   */
  status: AppStatus;
  /**
   * The Stripe account ID associated with the connected Stripe account.
   */
  accountId: string;
  /**
   * Indicates whether the app is connected to a live Stripe account.
   */
  livemode: boolean;
  /**
   * The masked Stripe API key that only exposes the first and last few characters.
   */
  maskedApiKey: string;
  /**
   * The Stripe secret API key used to authenticate API requests.
   */
  secretApiKey?: string;
}
/**
 * The type of the app.
 */
export enum AppType {
  /**
   * Built-in sandbox integration for testing and development.
   */
  Sandbox = "sandbox",
  /**
   * The Stripe app synchronizes invoices to Stripe Invoicing, enabling automated revenue collection with Stripe Payments and Stripe Tax.
   */
  Stripe = "stripe",
  /**
   * The external invoicing app enables synchronizing invoices with finance systems that are not natively supported, such as ERPs, in-house invoicing solutions, or local e-invoicing and payment providers.
   */
  ExternalInvoicing = "external_invoicing"
}
/**
 * Available apps for billing integrations to connect with third-party services.
 * Apps can have various capabilities like syncing data from or to external
 * systems, integrating with third-party services for tax calculation, delivery of
 * invoices, collection of payments, etc.
 */
export interface AppCatalogItem {
  /**
   * Type of the app.
   */
  type: AppType;
  /**
   * Name of the app.
   */
  name: string;
  /**
   * Description of the app.
   */
  description: string;
}
/**
 * Connection status of an installed app.
 */
export enum AppStatus {
  /**
   * The app is ready to be used.
   */
  Ready = "ready",
  /**
   * The app is unauthorized.
   * This usually happens when the app's credentials are revoked or expired.
   * To resolve this, the user must re-authorize the app.
   */
  Unauthorized = "unauthorized"
}
/**
 * Sandbox app can be used for testing billing features.
 */
export interface AppSandbox {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  /**
   * The app type.
   */
  type: AppType.Sandbox;
  /**
   * The app catalog definition that this installed app is based on.
   */
  definition: AppCatalogItem;
  /**
   * Status of the app connection.
   */
  status: AppStatus;
}
/**
 * External Invoicing app enables integration with third-party invoicing or payment
 * system.
 *
 * The app supports a bi-directional synchronization pattern where OpenMeter
 * Billing manages the invoice lifecycle while the external system handles invoice
 * presentation and payment collection.
 *
 * Integration workflow:
 *
 * 1. The billing system creates invoices and transitions them through lifecycle
 * states (draft → issuing → issued)
 * 2. The integration receives webhook notifications about invoice state changes
 * 3. The integration calls back to provide external system IDs and metadata
 * 4. The integration reports payment events back via the payment status API
 *
 * State synchronization is controlled by hooks that pause invoice progression
 * until the external system confirms synchronization via API callbacks.
 */
export interface AppExternalInvoicing {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  /**
   * The app type.
   */
  type: AppType.ExternalInvoicing;
  /**
   * The app catalog definition that this installed app is based on.
   */
  definition: AppCatalogItem;
  /**
   * Status of the app connection.
   */
  status: AppStatus;
  /**
   * Enable draft synchronization hook.
   *
   * When enabled, invoices will pause at the draft state and wait for the
   * integration to call the draft synchronized endpoint before progressing to the
   * issuing state. This allows the external system to validate and prepare the
   * invoice data.
   *
   * When disabled, invoices automatically progress through the draft state based on
   * the configured workflow timing.
   */
  enableDraftSyncHook: boolean;
  /**
   * Enable issuing synchronization hook.
   *
   * When enabled, invoices will pause at the issuing state and wait for the
   * integration to call the issuing synchronized endpoint before progressing to the
   * issued state. This ensures the external invoicing system has successfully
   * created and finalized the invoice before it is marked as issued.
   *
   * When disabled, invoices automatically progress through the issuing state and are
   * immediately marked as issued.
   */
  enableIssuingSyncHook: boolean;
}
/**
 * Billing profiles contain the settings for billing and controls invoice
 * generation.
 */
export interface BillingProfile {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  /**
   * The name and contact information for the supplier this billing profile
   * represents
   */
  supplier: BillingParty;
  /**
   * The billing workflow settings for this profile
   */
  workflow: BillingWorkflow;
  /**
   * The applications used by this billing profile.
   */
  apps: BillingProfileAppReferences;
  /**
   * Whether this is the default profile.
   */
  default_: boolean;
}
/**
 * Party represents a person or business entity.
 */
export interface BillingParty {
  /**
   * Unique identifier for the party.
   */
  id?: string;
  /**
   * An optional unique key of the party.
   */
  key?: string;
  /**
   * Legal name or representation of the party.
   */
  name?: string;
  /**
   * The entity's legal identification used for tax purposes. They may have other
   * numbers, but we're only interested in those valid for tax purposes.
   */
  taxId?: BillingPartyTaxIdentity;
  /**
   * Address for where information should be sent if needed.
   */
  addresses?: BillingPartyAddresses;
}
/**
 * Identity stores the details required to identify an entity for tax purposes in a
 * specific country.
 */
export interface BillingPartyTaxIdentity {
  /**
   * Normalized tax identification code shown on the original identity document.
   */
  code?: string;
}
/**
 * Tax identifier code is a normalized tax code shown on the original identity
 * document.
 */
export type TaxIdentificationCode = string;
/**
 * A collection of addresses for the party.
 */
export interface BillingPartyAddresses {
  /**
   * Billing address.
   */
  billingAddress: Address_2;
}
/**
 * Address
 */
export interface Address_2 {
  /**
   * Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html)
   * alpha-2 format.
   */
  country?: string;
  /**
   * Postal code.
   */
  postalCode?: string;
  /**
   * State or province.
   */
  state?: string;
  /**
   * City.
   */
  city?: string;
  /**
   * First line of the address.
   */
  line1?: string;
  /**
   * Second line of the address.
   */
  line2?: string;
  /**
   * Phone number.
   */
  phoneNumber?: string;
}
/**
 * Billing workflow settings.
 */
export interface BillingWorkflow {
  /**
   * The collection settings for this workflow
   */
  collection?: BillingWorkflowCollectionSettings;
  /**
   * The invoicing settings for this workflow
   */
  invoicing?: BillingWorkflowInvoicingSettings;
  /**
   * The payment settings for this workflow
   */
  payment?: BillingWorkflowPaymentSettings;
  /**
   * The tax settings for this workflow
   */
  tax?: BillingWorkflowTaxSettings;
}
/**
 * Workflow collection specifies how to collect the pending line items for an
 * invoice.
 */
export interface BillingWorkflowCollectionSettings {
  /**
   * The alignment for collecting the pending line items into an invoice.
   */
  alignment?: BillingWorkflowCollectionAlignment;
  /**
   * This grace period can be used to delay the collection of the pending line items
   * specified in alignment.
   *
   * This is useful, in case of multiple subscriptions having slightly different
   * billing periods.
   */
  interval?: string;
}
/**
 * The alignment for collecting the pending line items into an invoice.
 *
 * Defaults to subscription, which means that we are to create a new invoice every
 * time the a subscription period starts (for in advance items) or ends (for in
 * arrears items).
 */
export type BillingWorkflowCollectionAlignment = {
  type: "subscription"
} & BillingWorkflowCollectionAlignmentSubscription | {
  type: "anchored"
} & BillingWorkflowCollectionAlignmentAnchored;
/**
 * BillingWorkflowCollectionAlignmentSubscription specifies the alignment for
 * collecting the pending line items into an invoice.
 */
export interface BillingWorkflowCollectionAlignmentSubscription {
  /**
   * The type of alignment.
   */
  type: BillingCollectionAlignmentType.Subscription;
}
/**
 * BillingCollectionAlignment specifies when the pending line items should be
 * collected into an invoice.
 */
export enum BillingCollectionAlignmentType {
  /**
   * Align the collection to the start of the subscription period.
   */
  Subscription = "subscription",
  /**
   * Align the collection to the anchor time and cadence.
   */
  Anchored = "anchored"
}
/**
 * BillingWorkflowCollectionAlignmentAnchored specifies the alignment for
 * collecting the pending line items into an invoice.
 */
export interface BillingWorkflowCollectionAlignmentAnchored {
  /**
   * The type of alignment.
   */
  type: BillingCollectionAlignmentType.Anchored;
  /**
   * The recurring period for the alignment.
   */
  recurringPeriod: RecurringPeriod;
}
/**
 * Recurring period with an anchor and an interval.
 */
export interface RecurringPeriod {
  /**
   * A date-time anchor to base the recurring period on.
   */
  anchor: Date;
  /**
   * The interval duration in ISO 8601 format.
   */
  interval: string;
}
/**
 * [ISO 8601 Duration](https://docs.digi.com/resources/documentation/digidocs/90001488-13/reference/r_iso_8601_duration_format.htm)
 * string.
 */
export type Iso8601Duration = string;
/**
 * Invoice settings for a billing workflow.
 */
export interface BillingWorkflowInvoicingSettings {
  /**
   * Whether to automatically issue the invoice after the draftPeriod has passed.
   */
  autoAdvance?: boolean;
  /**
   * The period for the invoice to be kept in draft status for manual reviews.
   */
  draftPeriod?: string;
  /**
   * Should progressive billing be allowed for this workflow?
   */
  progressiveBilling?: boolean;
}
/**
 * Payment settings for a billing workflow.
 */
export type BillingWorkflowPaymentSettings = {
  collection_method: "charge_automatically"
} & BillingWorkflowPaymentChargeAutomaticallySettings | {
  collection_method: "send_invoice"
} & BillingWorkflowPaymentSendInvoiceSettings;
/**
 * Payment settings for a billing workflow when the collection method is charge
 * automatically.
 */
export interface BillingWorkflowPaymentChargeAutomaticallySettings {
  /**
   * The collection method for the invoice.
   */
  collectionMethod: CollectionMethod.ChargeAutomatically;
}
/**
 * Collection method specifies how the invoice should be collected (automatic or
 * manual).
 */
export enum CollectionMethod {
  ChargeAutomatically = "charge_automatically",
  SendInvoice = "send_invoice"
}
/**
 * Payment settings for a billing workflow when the collection method is send
 * invoice.
 */
export interface BillingWorkflowPaymentSendInvoiceSettings {
  /**
   * The collection method for the invoice.
   */
  collectionMethod: CollectionMethod.SendInvoice;
  /**
   * The period after which the invoice is due. With some payment solutions it's only
   * applicable for manual collection method.
   */
  dueAfter?: string;
}
/**
 * Tax settings for a billing workflow.
 */
export interface BillingWorkflowTaxSettings {
  /**
   * Enable automatic tax calculation when tax is supported by the app. For example,
   * with Stripe Invoicing when enabled, tax is calculated via Stripe Tax.
   */
  enabled?: boolean;
  /**
   * Enforce tax calculation when tax is supported by the app. When enabled, the
   * billing system will not allow to create an invoice without tax calculation.
   * Enforcement is different per apps, for example, Stripe app requires customer to
   * have a tax location when starting a paid subscription.
   */
  enforced?: boolean;
  /**
   * Default tax configuration to apply to the invoices for line items.
   */
  defaultTaxConfig?: TaxConfig;
}
/**
 * References to the applications used by a billing profile.
 */
export interface BillingProfileAppReferences {
  /**
   * The tax app used for this workflow.
   */
  tax: AppReference;
  /**
   * The invoicing app used for this workflow.
   */
  invoicing: AppReference;
  /**
   * The payment app used for this workflow.
   */
  payment: AppReference;
}
/**
 * App reference.
 */
export interface AppReference {
  /**
   * The ID of the app.
   */
  id: string;
}
/**
 * BillingProfile create request.
 */
export interface CreateRequest_4 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * The name and contact information for the supplier this billing profile
   * represents
   */
  supplier: BillingParty;
  /**
   * The billing workflow settings for this profile
   */
  workflow: BillingWorkflow;
  /**
   * The applications used by this billing profile.
   */
  apps: BillingProfileAppReferences;
  /**
   * Whether this is the default profile.
   */
  default_: boolean;
}
/**
 * BillingProfile upsert request.
 */
export interface UpsertRequest_4 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * The name and contact information for the supplier this billing profile
   * represents
   */
  supplier: BillingParty;
  /**
   * The billing workflow settings for this profile
   */
  workflow: BillingWorkflow;
  /**
   * Whether this is the default profile.
   */
  default_: boolean;
}
/**
 * TaxCode create request.
 */
export interface CreateRequest_5 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  key: string;
  /**
   * Mapping of app types to tax codes.
   */
  appMappings: Array<TaxCodeAppMapping>;
}
/**
 * Mapping of app types to tax codes.
 */
export interface TaxCodeAppMapping {
  /**
   * The app type that the tax code is associated with.
   */
  appType: AppType;
  /**
   * Tax code.
   */
  taxCode: string;
}
/**
 * Tax codes by provider.
 */
export interface TaxCode {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  key: string;
  /**
   * Mapping of app types to tax codes.
   */
  appMappings: Array<TaxCodeAppMapping>;
}
/**
 * TaxCode upsert request.
 */
export interface UpsertRequest_5 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * Mapping of app types to tax codes.
   */
  appMappings: Array<TaxCodeAppMapping>;
}
/**
 * Filter options for listing currencies.
 */
export interface ListCurrenciesParamsFilter {
  /**
   * Filter currencies by type.
   */
  type?: CurrencyType;
}
/**
 * Currency type for custom currencies. It should be a unique code but not
 * conflicting with any existing standard currency codes.
 */
export enum CurrencyType {
  Fiat = "fiat",
  Custom = "custom"
}
/**
 * Fiat or custom currency.
 */
export type Currency = {
  type: "fiat"
} & CurrencyFiat | {
  type: "custom"
} & CurrencyCustom;
/**
 * Currency describes a currency supported by the billing system.
 */
export interface CurrencyFiat {
  id: string;
  /**
   * The type of the currency.
   */
  type: CurrencyType.Fiat;
  /**
   * The name of the currency. It should be a human-readable string that represents
   * the name of the currency, such as "US Dollar" or "Euro".
   */
  name: string;
  /**
   * Description of the currency.
   */
  description?: string;
  /**
   * The symbol of the currency. It should be a string that represents the symbol of
   * the currency, such as "$" for US Dollar or "€" for Euro.
   */
  symbol?: string;
  code: string;
}
/**
 * Describes custom currency.
 */
export interface CurrencyCustom {
  id: string;
  /**
   * The type of the currency.
   */
  type: CurrencyType.Custom;
  /**
   * The name of the currency. It should be a human-readable string that represents
   * the name of the currency, such as "US Dollar" or "Euro".
   */
  name: string;
  /**
   * Description of the currency.
   */
  description?: string;
  /**
   * The symbol of the currency. It should be a string that represents the symbol of
   * the currency, such as "$" for US Dollar or "€" for Euro.
   */
  symbol?: string;
  code: string;
  /**
   * An ISO-8601 timestamp representation of the custom currency creation date.
   */
  createdAt: Date;
}
/**
 * Custom currency code. It should be a unique code but not conflicting with any
 * existing fiat currency codes.
 */
export type CurrencyCodeCustom = string;
/**
 * CurrencyCustom create request.
 */
export interface CreateRequest_6 {
  /**
   * The name of the currency. It should be a human-readable string that represents
   * the name of the currency, such as "US Dollar" or "Euro".
   */
  name: string;
  /**
   * Description of the currency.
   */
  description?: string;
  /**
   * The symbol of the currency. It should be a string that represents the symbol of
   * the currency, such as "$" for US Dollar or "€" for Euro.
   */
  symbol?: string;
  code: string;
}
/**
 * Filter options for listing cost bases.
 */
export interface ListCostBasesParamsFilter {
  /**
   * Filter cost bases by fiat currency code.
   */
  fiatCode?: string;
}
/**
 * Describes currency basis supported by billing system.
 */
export interface CostBasis {
  id: string;
  /**
   * The fiat currency code for the cost basis.
   */
  fiatCode: string;
  /**
   * The cost rate for the currency.
   */
  rate: string;
  /**
   * An ISO-8601 timestamp representation of the date from which the cost basis is
   * effective. If not provided, it will be effective immediately and will be set to
   * `now` by the system.
   */
  effectiveFrom?: Date;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
}
/**
 * CostBasis create request.
 */
export interface CreateRequest_7 {
  /**
   * The fiat currency code for the cost basis.
   */
  fiatCode: string;
  /**
   * The cost rate for the currency.
   */
  rate: string;
  /**
   * An ISO-8601 timestamp representation of the date from which the cost basis is
   * effective. If not provided, it will be effective immediately and will be set to
   * `now` by the system.
   */
  effectiveFrom?: Date;
}
/**
 * Filter options for listing features.
 */
export interface ListFeaturesParamsFilter {
  meterId?: UlidFieldFilter;
  key?: StringFieldFilter;
  name?: StringFieldFilter;
}
/**
 * A capability or billable dimension offered by a provider.
 */
export interface Feature {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  key: string;
  /**
   * The meter that the feature is associated with and based on which usage is
   * calculated. If not specified, the feature is static.
   */
  meter?: FeatureMeterReference;
  /**
   * Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or
   * "llm" to look up cost from the LLM cost database based on meter group-by
   * properties.
   */
  unitCost?: FeatureUnitCost;
}
/**
 * Reference to a meter associated with a feature.
 */
export interface FeatureMeterReference {
  /**
   * The ID of the meter to associate with this feature.
   */
  id: string;
  /**
   * Filters to apply to the dimensions of the meter.
   */
  filters?: Record<string, QueryFilterStringMapItem>;
}
/**
 * Per-unit cost configuration for a feature. Either a fixed manual amount or a
 * dynamic LLM cost lookup.
 */
export type FeatureUnitCost = {
  type: "manual"
} & FeatureManualUnitCost | {
  type: "llm"
} & FeatureLlmUnitCost;
/**
 * A fixed per-unit cost amount.
 */
export interface FeatureManualUnitCost {
  /**
   * The type discriminator for manual unit cost.
   */
  type: FeatureUnitCostType.Manual;
  /**
   * Fixed per-unit cost amount in USD.
   */
  amount: string;
}
/**
 * The type of unit cost.
 */
export enum FeatureUnitCostType {
  Llm = "llm",
  Manual = "manual"
}
/**
 * LLM cost lookup configuration. Each dimension (provider, model, token type) can
 * be specified as either a static value or a meter group-by property name
 * (mutually exclusive).
 */
export interface FeatureLlmUnitCost {
  /**
   * The type discriminator for LLM unit cost.
   */
  type: FeatureUnitCostType.Llm;
  /**
   * Meter group-by property that holds the LLM provider. Use this when the meter has
   * a group-by dimension for provider. Mutually exclusive with `provider`.
   */
  providerProperty?: string;
  /**
   * Static LLM provider value (e.g., "openai", "anthropic"). Use this when the
   * feature tracks a single provider. Mutually exclusive with `provider_property`.
   */
  provider?: string;
  /**
   * Meter group-by property that holds the model ID. Use this when the meter has a
   * group-by dimension for model. Mutually exclusive with `model`.
   */
  modelProperty?: string;
  /**
   * Static model ID value (e.g., "gpt-4", "claude-3-5-sonnet"). Use this when the
   * feature tracks a single model. Mutually exclusive with `model_property`.
   */
  model?: string;
  /**
   * Meter group-by property that holds the token type. Use this when the meter has a
   * group-by dimension for token type. Mutually exclusive with `token_type`.
   */
  tokenTypeProperty?: string;
  /**
   * Static token type value. Use this when the feature tracks a single token type
   * (e.g., only input tokens). `request` is an alias for `input`, `response` is an
   * alias for `output`. Mutually exclusive with `token_type_property`.
   */
  tokenType?: FeatureLlmTokenType;
  /**
   * Resolved per-token pricing from the LLM cost database. Populated in responses
   * when the provider and model can be determined, either from static values or from
   * meter group-by filters with exact matches.
   */
  pricing?: FeatureLlmUnitCostPricing;
}
/**
 * Token type for LLM cost lookup.
 */
export enum FeatureLlmTokenType {
  Input = "input",
  Output = "output",
  CacheRead = "cache_read",
  CacheWrite = "cache_write",
  Reasoning = "reasoning",
  /**
   * Alias for `input`.
   */
  Request = "request",
  /**
   * Alias for `output`.
   */
  Response = "response"
}
/**
 * Resolved per-token pricing from the LLM cost database.
 */
export interface FeatureLlmUnitCostPricing {
  /**
   * Cost per input token in USD.
   */
  inputPerToken: string;
  /**
   * Cost per output token in USD.
   */
  outputPerToken: string;
  /**
   * Cost per cache read token in USD.
   */
  cacheReadPerToken?: string;
  /**
   * Cost per reasoning token in USD.
   */
  reasoningPerToken?: string;
  /**
   * Cost per cache write token in USD.
   */
  cacheWritePerToken?: string;
}
/**
 * Feature create request.
 */
export interface CreateRequest_8 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  key: string;
  /**
   * The meter that the feature is associated with and based on which usage is
   * calculated. If not specified, the feature is static.
   */
  meter?: FeatureMeterReference;
  /**
   * Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or
   * "llm" to look up cost from the LLM cost database based on meter group-by
   * properties.
   */
  unitCost?: FeatureUnitCost;
}
/**
 * Request body for updating a feature. Currently only the unit_cost field can be
 * updated.
 */
export interface FeatureUpdateRequest {
  /**
   * Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or
   * "llm" to look up cost from the LLM cost database based on meter group-by
   * properties. Set to `null` to clear the existing unit cost; omit to leave it
   * unchanged.
   */
  unitCost?: FeatureUnitCost | null;
}
/**
 * Result of a feature cost query.
 */
export interface FeatureCostQueryResult {
  /**
   * Start of the queried period.
   */
  from?: Date;
  /**
   * End of the queried period.
   */
  to?: Date;
  /**
   * The cost data rows.
   */
  data: Array<FeatureCostQueryRow>;
}
/**
 * A row in the result of a feature cost query.
 */
export interface FeatureCostQueryRow {
  /**
   * The metered usage value for the period.
   */
  usage: string;
  /**
   * The computed cost amount (usage × unit cost). Null when pricing is not available
   * for the given combination of dimensions.
   */
  cost: string | null;
  /**
   * The currency code of the cost amount.
   */
  currency: string;
  /**
   * Detail message when cost amount is null, explaining why the cost could not be
   * resolved.
   */
  detail?: string;
  /**
   * The start of the time bucket the value is aggregated over.
   */
  from: Date;
  /**
   * The end of the time bucket the value is aggregated over.
   */
  to: Date;
  /**
   * The dimensions the value is aggregated over. `subject` and `customer_id` are
   * reserved dimensions.
   */
  dimensions: Record<string, string>;
}
/**
 * Filter options for listing LLM cost prices.
 */
export interface ListPricesParamsFilter {
  /**
   * Filter by provider. e.g. ?filter[provider][eq]=openai
   */
  provider?: StringFieldFilter;
  /**
   * Filter by model ID. e.g. ?filter[model_id][eq]=gpt-4
   */
  modelId?: StringFieldFilter;
  /**
   * Filter by model name. e.g. ?filter[model_name][contains]=gpt
   */
  modelName?: StringFieldFilter;
  /**
   * Filter by currency code. e.g. ?filter[currency][eq]=USD
   */
  currency?: StringFieldFilter;
  /**
   * Filter by source. e.g. ?filter[source][eq]=system
   */
  source?: StringFieldFilter;
}
/**
 * An LLM cost price record, representing the cost per token for a specific model
 * from a specific provider.
 */
export interface Price_2 {
  /**
   * Unique identifier.
   */
  id: string;
  /**
   * Provider of the model.
   */
  provider: Provider;
  /**
   * The model.
   */
  model: Model;
  /**
   * Token pricing data.
   */
  pricing: ModelPricing;
  /**
   * Currency code (currently always "USD").
   */
  currency: string;
  /**
   * Where this price came from.
   */
  source: PriceSource;
  /**
   * When this price becomes effective.
   */
  effectiveFrom: Date;
  /**
   * When this price expires. Omitted when the price is currently effective.
   */
  effectiveTo?: Date;
  /**
   * Creation timestamp.
   */
  createdAt: Date;
  /**
   * Last update timestamp.
   */
  updatedAt: Date;
}
/**
 * LLM Provider
 */
export interface Provider {
  /**
   * Identifier of the provider, e.g., "openai", "anthropic".
   */
  id: string;
  /**
   * Name of the provider, e.g., "OpenAI", "Anthropic".
   */
  name: string;
}
/**
 * LLM Model
 */
export interface Model {
  /**
   * Identifier of the model, e.g., "gpt-4", "claude-3-5-sonnet".
   */
  id: string;
  /**
   * Name of the model, e.g., "GPT-4", "Claude 3.5 Sonnet".
   */
  name: string;
}
/**
 * Token pricing for an LLM model, denominated per token.
 */
export interface ModelPricing {
  /**
   * Input price per token (USD).
   */
  inputPerToken: string;
  /**
   * Output price per token (USD).
   */
  outputPerToken: string;
  /**
   * Cache read price per token (USD).
   */
  cacheReadPerToken?: string;
  /**
   * Cache write price per token (USD).
   */
  cacheWritePerToken?: string;
  /**
   * Reasoning output price per token (USD).
   */
  reasoningPerToken?: string;
}
/**
 * Identifies where an LLM cost price came from.
 */
export enum PriceSource {
  /**
   * Price was manually configured by a user.
   */
  Manual = "manual",
  /**
   * Price was automatically synced from external pricing sources.
   */
  System = "system"
}
/**
 * Input for creating a per-namespace price override. Unique per provider, model
 * and currency. If an override already exists for the given provider, model and
 * currency, it will be updated. If an override does not exist, it will be created.
 */
export interface OverrideCreate {
  /**
   * Provider/vendor of the model.
   */
  provider: string;
  /**
   * Canonical model identifier.
   */
  modelId: string;
  /**
   * Human-readable model name.
   */
  modelName?: string;
  /**
   * Token pricing data.
   */
  pricing: ModelPricing;
  /**
   * Currency code.
   */
  currency: string;
  /**
   * When this override becomes effective.
   */
  effectiveFrom: Date;
  /**
   * When this override expires.
   */
  effectiveTo?: Date;
}
/**
 * Filter options for listing plans.
 */
export interface ListPlansParamsFilter {
  key?: StringFieldFilter;
  name?: StringFieldFilter;
  status?: StringFieldFilterExact;
  currency?: StringFieldFilterExact;
}
/**
 * Plans provide a template for subscriptions.
 */
export interface Plan {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  /**
   * A key is a semi-unique string that is used to identify the plan. It is used to
   * reference the latest `active` version of the plan and is unique with the version
   * number.
   */
  key: string;
  /**
   * Plans are versioned to allow you to make changes without affecting running
   * subscriptions.
   */
  version: number;
  /**
   * The currency code of the plan.
   */
  currency: string;
  /**
   * The billing cadence for subscriptions using this plan.
   */
  billingCadence: string;
  /**
   * Whether pro-rating is enabled for this plan.
   */
  proRatingEnabled?: boolean;
  /**
   * The date and time when the plan becomes `active`. When not specified, the plan
   * is in `draft` status.
   */
  effectiveFrom?: Date;
  /**
   * A scheduled date and time when the plan becomes `archived`. When not specified,
   * the plan is in `active` status indefinitely.
   */
  effectiveTo?: Date;
  /**
   * The status of the plan. Computed based on the effective start and end dates:
   *
   * - `draft`: `effective_from` is not set.
   * - `scheduled`: `now < effective_from`.
   * - `active`: `effective_from <= now` and (`effective_to` is not set or
   * `now < effective_to`).
   * - `archived`: `effective_to <= now`.
   */
  status: PlanStatus;
  /**
   * The plan phases define the pricing ramp for a subscription. A phase switch
   * occurs only at the end of a billing period. At least one phase is required.
   */
  phases: Array<PlanPhase>;
  /**
   * List of validation errors in `draft` state that prevent the plan from being
   * published.
   */
  validationErrors?: Array<ProductCatalogValidationError>;
}
/**
 * The status of a plan.
 *
 * - `draft`: The plan has not yet been published and can be edited.
 * - `active`: The plan is published and can be used in subscriptions.
 * - `archived`: The plan is no longer available for use.
 * - `scheduled`: The plan is scheduled to be published at a future date.
 */
export enum PlanStatus {
  Draft = "draft",
  Active = "active",
  Archived = "archived",
  Scheduled = "scheduled"
}
/**
 * The plan phase or pricing ramp allows changing a plan's rate cards over time as
 * a subscription progresses.
 */
export interface PlanPhase {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  key: string;
  /**
   * The duration of the phase. When not specified, the phase runs indefinitely. Only
   * the last phase may omit the duration.
   */
  duration?: string;
  /**
   * The rate cards of the plan.
   */
  rateCards: Array<RateCard>;
}
/**
 * A rate card defines the pricing and entitlement of a feature or service.
 */
export interface RateCard {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  key: string;
  /**
   * The feature associated with the rate card.
   */
  feature?: ResourceReference_4;
  /**
   * The billing cadence of the rate card. When null, the charge is one-time
   * (non-recurring). Only valid for flat prices.
   */
  billingCadence?: string;
  /**
   * The price of the rate card.
   */
  price: Price;
  /**
   * The payment term of the rate card. In advance payment term can only be used for
   * flat prices.
   */
  paymentTerm?: PricePaymentTerm;
  /**
   * Spend commitments for this rate card. Only applicable to usage-based prices
   * (unit, graduated, volume).
   */
  commitments?: SpendCommitments;
  /**
   * The discounts of the rate card.
   */
  discounts?: Discounts;
  /**
   * The tax config of the rate card.
   */
  taxConfig?: RateCardTaxConfig;
}
/**
 * Feature reference.
 */
export interface ResourceReference_4 {
  id: string;
}
/**
 * Spend commitments for a rate card. The customer is committed to spend at least
 * the minimum amount and at most the maximum amount.
 */
export interface SpendCommitments {
  /**
   * The customer is committed to spend at least the amount.
   */
  minimumAmount?: string;
  /**
   * The customer is limited to spend at most the amount.
   */
  maximumAmount?: string;
}
/**
 * The tax config of the rate card.
 */
export interface RateCardTaxConfig {
  behavior?: TaxBehavior;
  code: ResourceReference_2;
}
/**
 * Validation errors providing detailed description of the issue.
 */
export interface ProductCatalogValidationError {
  /**
   * Machine-readable error code.
   */
  code: string;
  /**
   * Human-readable description of the error.
   */
  message: string;
  /**
   * Additional structured context.
   */
  attributes?: Record<string, unknown>;
  /**
   * The path to the field.
   */
  field: string;
}
/**
 * Plan create request.
 */
export interface CreateRequest_9 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * A key is a semi-unique string that is used to identify the plan. It is used to
   * reference the latest `active` version of the plan and is unique with the version
   * number.
   */
  key: string;
  /**
   * The currency code of the plan.
   */
  currency: string;
  /**
   * The billing cadence for subscriptions using this plan.
   */
  billingCadence: string;
  /**
   * Whether pro-rating is enabled for this plan.
   */
  proRatingEnabled?: boolean;
  /**
   * The plan phases define the pricing ramp for a subscription. A phase switch
   * occurs only at the end of a billing period. At least one phase is required.
   */
  phases: Array<PlanPhase>;
}
/**
 * Plan upsert request.
 */
export interface UpsertRequest_6 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * Whether pro-rating is enabled for this plan.
   */
  proRatingEnabled?: boolean;
  /**
   * The plan phases define the pricing ramp for a subscription. A phase switch
   * occurs only at the end of a billing period. At least one phase is required.
   */
  phases: Array<PlanPhase>;
}
/**
 * Filter options for listing add-ons.
 */
export interface ListAddonsParamsFilter {
  id?: UlidFieldFilter;
  key?: StringFieldFilter;
  name?: StringFieldFilter;
  status?: StringFieldFilterExact;
  currency?: StringFieldFilterExact;
}
/**
 * Add-on allows extending subscriptions with compatible plans with additional
 * ratecards.
 */
export interface Addon {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  /**
   * A key is a semi-unique string that is used to identify the add-on. It is used to
   * reference the latest `active` version of the add-on and is unique with the
   * version number.
   */
  key: string;
  /**
   * Version of the add-on. Incremented when the add-on is updated.
   */
  version: number;
  /**
   * The InstanceType of the add-ons. Can be "single" or "multiple".
   */
  instanceType: AddonInstanceType;
  /**
   * The currency code of the add-on.
   */
  currency: CurrencyCode_2;
  /**
   * The date and time when the add-on becomes effective. When not specified, the
   * add-on is a draft.
   */
  effectiveFrom?: Date;
  /**
   * The date and time when the add-on is no longer effective. When not specified,
   * the add-on is effective indefinitely.
   */
  effectiveTo?: Date;
  /**
   * The status of the add-on. Computed based on the effective start and end dates:
   *
   * - `draft`: `effective_from` is not set.
   * - `active`: `effective_from <= now` and (`effective_to` is not set or
   * `now < effective_to`).
   * - `archived`: `effective_to <= now`.
   */
  status: AddonStatus;
  /**
   * The rate cards of the add-on.
   */
  rateCards: Array<RateCard>;
  /**
   * List of validation errors.
   */
  validationErrors?: Array<ProductCatalogValidationError>;
}
/**
 * The instanceType of the add-on.
 *
 * - `single`: Can be added to a subscription only once.
 * - `multiple`: Can be added to a subscription more than once.
 */
export enum AddonInstanceType {
  Single = "single",
  Multiple = "multiple"
}
/**
 * The status of the add-on defined by the `effective_from` and `effective_to`
 * properties.
 *
 * - `draft`: The add-on has not yet been published and can be edited.
 * - `active`: The add-on is published and available for use.
 * - `archived`: The add-on is no longer available for use.
 */
export enum AddonStatus {
  Draft = "draft",
  Active = "active",
  Archived = "archived"
}
/**
 * Addon create request.
 */
export interface CreateRequest_10 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * A key is a semi-unique string that is used to identify the add-on. It is used to
   * reference the latest `active` version of the add-on and is unique with the
   * version number.
   */
  key: string;
  /**
   * The InstanceType of the add-ons. Can be "single" or "multiple".
   */
  instanceType: AddonInstanceType;
  /**
   * The currency code of the add-on.
   */
  currency: CurrencyCode_2;
  /**
   * The rate cards of the add-on.
   */
  rateCards: Array<RateCard>;
}
/**
 * Addon upsert request.
 */
export interface UpsertRequest_7 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * The InstanceType of the add-ons. Can be "single" or "multiple".
   */
  instanceType: AddonInstanceType;
  /**
   * The rate cards of the add-on.
   */
  rateCards: Array<RateCard>;
}
/**
 * PlanAddon represents an association between a plan and an add-on, controlling
 * which add-ons are available for purchase within a plan.
 */
export interface PlanAddon {
  id: string;
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * An ISO-8601 timestamp representation of entity creation date.
   */
  createdAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity last update date.
   */
  updatedAt: Date;
  /**
   * An ISO-8601 timestamp representation of entity deletion date.
   */
  deletedAt?: Date;
  /**
   * The add-on associated with the plan.
   */
  addon: ResourceReference_3;
  /**
   * The key of the plan phase from which the add-on becomes available for purchase.
   */
  fromPlanPhase: string;
  /**
   * The maximum number of times the add-on can be purchased for the plan. For
   * single-instance add-ons this field must be omitted. For multi-instance add-ons
   * when omitted, unlimited quantity can be purchased.
   */
  maxQuantity?: number;
  /**
   * List of validation errors.
   */
  validationErrors?: Array<ProductCatalogValidationError>;
}
/**
 * PlanAddon create request.
 */
export interface CreateRequest_11 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * The add-on associated with the plan.
   */
  addon: ResourceReference_3;
  /**
   * The key of the plan phase from which the add-on becomes available for purchase.
   */
  fromPlanPhase: string;
  /**
   * The maximum number of times the add-on can be purchased for the plan. For
   * single-instance add-ons this field must be omitted. For multi-instance add-ons
   * when omitted, unlimited quantity can be purchased.
   */
  maxQuantity?: number;
}
/**
 * PlanAddon upsert request.
 */
export interface UpsertRequest_8 {
  /**
   * Display name of the resource.
   *
   * Between 1 and 256 characters.
   */
  name: string;
  /**
   * Optional description of the resource.
   *
   * Maximum 1024 characters.
   */
  description?: string;
  labels?: Labels;
  /**
   * The key of the plan phase from which the add-on becomes available for purchase.
   */
  fromPlanPhase: string;
  /**
   * The maximum number of times the add-on can be purchased for the plan. For
   * single-instance add-ons this field must be omitted. For multi-instance add-ons
   * when omitted, unlimited quantity can be purchased.
   */
  maxQuantity?: number;
}
/**
 * Organization-level default tax code references.
 *
 * Stores the default tax codes applied to specific billing contexts for this
 * organization. Provisioned automatically when the organization is created.
 */
export interface OrganizationDefaultTaxCodes {
  /**
   * Default tax code for invoicing.
   */
  invoicingTaxCode: ResourceReference_2;
  /**
   * Default tax code for credit grants.
   */
  creditGrantTaxCode: ResourceReference_2;
  /**
   * Timestamp of creation.
   */
  createdAt: Date;
  /**
   * Timestamp of last update.
   */
  updatedAt: Date;
}
/**
 * OrganizationDefaultTaxCodes update request.
 */
export interface UpdateRequest_2 {
  /**
   * Default tax code for invoicing.
   */
  invoicingTaxCode?: ResourceReference_2;
  /**
   * Default tax code for credit grants.
   */
  creditGrantTaxCode?: ResourceReference_2;
}
/**
 * Query to evaluate feature access for a list of customers.
 */
export interface GovernanceQueryRequest {
  /**
   * Whether to include credit balance availability for each resolved customer. When
   * true, each feature evaluation includes credit balance checks.
   *
   * Defaults to `false`.
   */
  includeCredits?: boolean;
  customer: GovernanceQueryRequestCustomers;
  feature?: GovernanceQueryRequestFeatures;
}
/**
 * List of customer identifiers to evaluate access for.
 */
export interface GovernanceQueryRequestCustomers {
  /**
   * Each entry can be a customer `key` or a usage-attribution subject `key`.
   * Identifiers that cannot be resolved to a customer are reported in the response
   * `errors` array.
   */
  keys: Array<string>;
}
/**
 * Optional list of feature keys to evaluate access for. If omitted, all features
 * available in the organization are returned. Providing this list is recommended
 * to reduce the response size and the load on the backend services.
 */
export interface GovernanceQueryRequestFeatures {
  /**
   * List of feature keys to evaluate access for.
   */
  keys: Array<string>;
}
/**
 * Response of the governance query.
 */
export interface GovernanceQueryResponse {
  /**
   * Access evaluation results, one entry per resolved customer.
   */
  data: Array<GovernanceQueryResult>;
  /**
   * Partial errors encountered while processing the request.
   */
  errors: Array<GovernanceQueryError>;
  /**
   * Pagination metadata. The endpoint may return a partial response if the full
   * response would exceed server-side limits.
   */
  meta: CursorMeta;
}
/**
 * Access evaluation result for a single resolved customer.
 */
export interface GovernanceQueryResult {
  /**
   * The list of identifiers from the request that resolved to this customer. Each
   * entry is either the customer `key` or one of its usage-attribution subject
   * `key`s.
   *
   * Duplicate or aliased identifiers that resolve to the same customer collapse to a
   * single result entry, with every requested identifier listed here.
   */
  matched: Array<string>;
  /**
   * The customer the matched identifiers resolved to.
   */
  customer: Customer;
  /**
   * Map of features with their access status.
   *
   * Map keys are the feature keys requested in `feature.keys`, or every feature
   * `key` available in the organization when the feature filter was omitted.
   */
  features: Record<string, GovernanceFeatureAccess>;
  /**
   * Timestamp of the most recent change to the customer's access state reflected in
   * this result.
   */
  updatedAt: Date;
}
/**
 * Access status for a single feature.
 */
export interface GovernanceFeatureAccess {
  /**
   * Whether the customer currently has access to the feature.
   *
   * `true` for boolean and static entitlements that are available, and for metered
   * entitlements with remaining balance. `false` when the feature is unavailable,
   * the usage limit has been reached, or (when applicable) credits have been
   * exhausted.
   */
  hasAccess: boolean;
  /**
   * Optional reason when the customer does not have access to the feature. Populated
   * when `has_access` is `false`.
   */
  reason?: GovernanceFeatureAccessReason;
}
/**
 * Reason a feature is not accessible to a customer.
 */
export interface GovernanceFeatureAccessReason {
  /**
   * Machine-readable error code.
   */
  code: GovernanceFeatureAccessReasonCode;
  /**
   * Human-readable description of the error.
   */
  message: string;
  /**
   * Additional structured context.
   */
  attributes?: Record<string, unknown>;
}
/**
 * Machine-readable reason code for denied feature access.
 */
export enum GovernanceFeatureAccessReasonCode {
  /**
   * Default zero value. Reserved for forward compatibility.
   */
  Unknown = "unknown",
  /**
   * The customer has reached the metered usage limit for the feature
   * within the current usage period.
   */
  UsageLimitReached = "usage_limit_reached",
  /**
   * The feature is not available to the customer.
   */
  FeatureUnavailable = "feature_unavailable",
  /**
   * The feature `key` referenced by the request is not configured in the
   * organization.
   */
  FeatureNotFound = "feature_not_found",
  /**
   * The customer has no available prepaid credit balance.
   */
  NoCreditAvailable = "no_credit_available"
}
/**
 * Query error within a partially successful governance query response.
 */
export interface GovernanceQueryError {
  /**
   * Machine-readable error code.
   */
  code: GovernanceQueryErrorCode;
  /**
   * Human-readable description of the error.
   */
  message: string;
  /**
   * Additional structured context.
   */
  attributes?: Record<string, unknown>;
  /**
   * The customer identifier from the request that produced this error.
   */
  customer?: string;
}
/**
 * Error code for a governance query failure.
 */
export enum GovernanceQueryErrorCode {
  /**
   * Default zero value. Reserved for forward compatibility.
   */
  Unknown = "unknown",
  /**
   * The provided identifier could not be resolved to any customer
   * (neither by customer `key` nor by a usage-attribution subject `key`).
   */
  CustomerNotFound = "customer_not_found"
}
/**
 * Field filters with all supported types.
 */
export interface FieldFilters {
  boolean?: BooleanFieldFilter;
  numeric?: NumericFieldFilter;
  string?: StringFieldFilter;
  stringExact?: StringFieldFilterExact;
  ulid?: UlidFieldFilter;
  datetime?: DateTimeFieldFilter;
  labels?: Record<string, StringFieldFilter>;
}
/**
 * Filter by a boolean value (true/false).
 */
export type BooleanFieldFilter = boolean | {
  /**
   * Value strictly equals the given boolean value.
   */
  eq: boolean;
};
/**
 * Filter by a numeric value. All properties are optional; provide exactly one to
 * specify the comparison.
 */
export type NumericFieldFilter = number | {
  /**
   * Value strictly equals the given numeric value.
   */
  eq?: number;
  /**
   * Value does not equal the given numeric value.
   */
  neq?: number;
  /**
   * Returns entities that match any of the comma-delimited numeric values.
   */
  oeq?: Array<number>;
  /**
   * Value is less than the given numeric value.
   */
  lt?: number;
  /**
   * Value is less than or equal to the given numeric value.
   */
  lte?: number;
  /**
   * Value is greater than the given numeric value.
   */
  gt?: number;
  /**
   * Value is greater than or equal to the given numeric value.
   */
  gte?: number;
};
/**
 * A 64 bit floating point number. (`±5.0 × 10^−324` to `±1.7 × 10^308`)
 */
export type Float64 = number;
