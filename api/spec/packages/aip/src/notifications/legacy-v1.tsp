import "@typespec/http";
import "@typespec/openapi";
import "@typespec/openapi3";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

/**
 * Copies of the legacy v1 API models carried by the legacy notification Event
 * payloads (invoice.created, invoice.updated, entitlements.balance.threshold and
 * entitlements.reset). The v3 API returns the same content as the v1 API, but with
 * snake_case field names. Map keys, such as those in metadata, are user data and
 * keep their case. Enum values keep their v1 spelling.
 *
 * The legacy TypeSpec program cannot be imported here, so this is a copy: change
 * it only to follow a change of the v1 payload.
 */
namespace Notifications.LegacyV1;

/**
 * Address
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1Address")
model V1Address {
  /**
   * Country code in [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html)
   * alpha-2 format.
   */
  country?: V1CountryCode;

  /**
   * Postal code.
   */
  postal_code?: string;

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
  phone_number?: string;
}

/**
 * Alignment configuration for a plan or subscription.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
#deprecated "deprecated"
@friendlyName("BillingNotificationEventV1Alignment")
model V1Alignment {
  /**
   * Whether all Billable items and RateCards must align. Alignment means the Price's
   * BillingCadence must align for both duration and anchor time.
   */
  #deprecated "deprecated"
  billables_must_align?: boolean;
}

/**
 * Set of key-value pairs managed by the system. Cannot be modified by user.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1Annotations")
model V1Annotations {
  ...Record<unknown>;
}

/**
 * App. One of: stripe
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
@oneOf
@friendlyName("BillingNotificationEventV1App")
union V1App {
  /**
   * stripe.
   */
  stripe: V1StripeApp,

  /**
   * sandbox.
   */
  sandbox: V1SandboxApp,

  /**
   * custom_invoicing.
   */
  custom_invoicing: V1CustomInvoicingApp,
}

/**
 * App capability.
 *
 * Capabilities only exist in config so they don't extend the Resource model.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1AppCapability")
model V1AppCapability {
  /**
   * The capability type.
   */
  type: V1AppCapabilityType;

  /**
   * Key
   */
  @minLength(1)
  @maxLength(64)
  @pattern("^[a-z0-9]+(?:_[a-z0-9]+)*$")
  key: string;

  /**
   * The capability name.
   */
  name: string;

  /**
   * The capability description.
   */
  description: string;
}

/**
 * App capability type.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1AppCapabilityType")
enum V1AppCapabilityType {
  reportUsage,
  reportEvents,
  calculateTax,
  invoiceCustomers,
  collectPayments,
}

/**
 * App installed status.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1AppStatus")
enum V1AppStatus {
  ready,
  unauthorized,
}

/**
 * Type of the app.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1AppType")
enum V1AppType {
  stripe,
  sandbox,
  custom_invoicing,
}

/**
 * A percentage discount.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingDiscountPercentage")
model V1BillingDiscountPercentage {
  /**
   * The percentage of the discount.
   */
  @minValue(0)
  @maxValue(100)
  @summary("Percentage")
  percentage: V1Percentage;

  /**
   * Correlation ID for the discount.
   *
   * This is used to link discounts across different invoices (progressive billing
   * use case).
   *
   * If not provided, the invoicing engine will auto-generate one. When editing an
   * invoice line, please make sure to keep the same correlation ID of the discount
   * or in progressive billing setups the discount amounts might be incorrect.
   */
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  correlation_id?: string;
}

/**
 * The reason for the discount.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
@oneOf
@friendlyName("BillingNotificationEventV1BillingDiscountReason")
union V1BillingDiscountReason {
  /**
   * maximum_spend.
   */
  maximum_spend: V1DiscountReasonMaximumSpend,

  /**
   * ratecard_percentage.
   */
  ratecard_percentage: V1DiscountReasonRatecardPercentage,

  /**
   * ratecard_usage.
   */
  ratecard_usage: V1DiscountReasonRatecardUsage,
}

/**
 * A usage discount.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingDiscountUsage")
model V1BillingDiscountUsage {
  /**
   * The quantity of the usage discount.
   *
   * Must be positive.
   */
  @summary("Usage") quantity: V1Numeric;

  /**
   * Correlation ID for the discount.
   *
   * This is used to link discounts across different invoices (progressive billing
   * use case).
   *
   * If not provided, the invoicing engine will auto-generate one. When editing an
   * invoice line, please make sure to keep the same correlation ID of the discount
   * or in progressive billing setups the discount amounts might be incorrect.
   */
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  correlation_id?: string;
}

/**
 * A discount by type.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingDiscounts")
model V1BillingDiscounts {
  /**
   * The percentage discount.
   */
  percentage?: V1BillingDiscountPercentage;

  /**
   * The usage discount.
   */
  usage?: V1BillingDiscountUsage;
}

/**
 * BillingInvoiceCustomerExtendedDetails is a collection of fields that are used to
 * extend the billing party details for invoices.
 *
 * These fields contain the OpenMeter specific details for the customer, that are
 * not strictly required for the invoice itself.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingInvoiceCustomerExtendedDetails")
model V1BillingInvoiceCustomerExtendedDetails {
  /**
   * Unique identifier for the party (if available)
   */
  @visibility(Lifecycle.Read) id?: string;

  /**
   * An optional unique key of the party (if available)
   */
  @minLength(1)
  @maxLength(256)
  @summary("Key")
  key?: string;

  /**
   * Legal name or representation of the organization.
   */
  name?: string;

  /**
   * The entity's legal ID code used for tax purposes. They may have other numbers,
   * but we're only interested in those valid for tax purposes.
   */
  tax_id?: V1BillingPartyTaxIdentity;

  /**
   * Regular post addresses for where information should be sent if needed.
   */
  @maxItems(1) addresses?: V1Address[];

  /**
   * Mapping to attribute metered usage to the customer
   */
  @summary("Usage Attribution") usage_attribution: V1CustomerUsageAttribution;
}

/**
 * Party represents a person or business entity.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingParty")
model V1BillingParty {
  /**
   * Unique identifier for the party (if available)
   */
  @visibility(Lifecycle.Read) id?: string;

  /**
   * An optional unique key of the party (if available)
   */
  @minLength(1)
  @maxLength(256)
  @summary("Key")
  key?: string;

  /**
   * Legal name or representation of the organization.
   */
  name?: string;

  /**
   * The entity's legal ID code used for tax purposes. They may have other numbers,
   * but we're only interested in those valid for tax purposes.
   */
  tax_id?: V1BillingPartyTaxIdentity;

  /**
   * Regular post addresses for where information should be sent if needed.
   */
  @maxItems(1) addresses?: V1Address[];
}

/**
 * Identity stores the details required to identify an entity for tax purposes in a
 * specific country.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingPartyTaxIdentity")
model V1BillingPartyTaxIdentity {
  /**
   * Normalized tax code shown on the original identity document.
   */
  code?: V1BillingTaxIdentificationCode;
}

/**
 * BillingProfileApps represents the applications used by a billing profile
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingProfileApps")
model V1BillingProfileApps {
  /**
   * The tax app used for this workflow
   */
  @visibility(Lifecycle.Read) tax: V1App;

  /**
   * The invoicing app used for this workflow
   */
  @visibility(Lifecycle.Read) invoicing: V1App;

  /**
   * The payment app used for this workflow
   */
  @visibility(Lifecycle.Read) payment: V1App;
}

/**
 * The settlement mode of a plan. It determines how the billing system generates
 * invoices and credits for the subscriptions using this plan.
 *
 * - credit_then_invoice: credits from the previous billing period are applied
 *   first, then the remaining balance is invoiced. This is the default and most
 *   common settlement mode.
 * - credit_only: only credits from the previous billing period are generated and
 *   applied. No invoices are generated for the subscription.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingSettlementMode")
enum V1BillingSettlementMode {
  credit_then_invoice,
  credit_only,
}

/**
 * TaxIdentificationCode is a normalized tax code shown on the original identity
 * document.
 */
@minLength(1)
@maxLength(32)
@friendlyName("BillingNotificationEventV1BillingTaxIdentificationCode")
scalar V1BillingTaxIdentificationCode extends string;

/**
 * BillingWorkflow represents the settings for a billing workflow.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingWorkflow")
model V1BillingWorkflow {
  /**
   * The collection settings for this workflow
   */
  collection?: V1BillingWorkflowCollectionSettings;

  /**
   * The invoicing settings for this workflow
   */
  invoicing?: V1BillingWorkflowInvoicingSettings;

  /**
   * The payment settings for this workflow
   */
  payment?: V1BillingWorkflowPaymentSettings;

  /**
   * The tax settings for this workflow
   */
  tax?: V1BillingWorkflowTaxSettings;
}

/**
 * The alignment for collecting the pending line items into an invoice.
 *
 * Defaults to subscription, which means that we are to create a new invoice every
 * time the a subscription period starts (for in advance items) or ends (for in
 * arrears items).
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
@oneOf
@friendlyName("BillingNotificationEventV1BillingWorkflowCollectionAlignment")
union V1BillingWorkflowCollectionAlignment {
  /**
   * subscription.
   */
  subscription: V1BillingWorkflowCollectionAlignmentSubscription,

  /**
   * anchored.
   */
  anchored: V1BillingWorkflowCollectionAlignmentAnchored,
}

/**
 * BillingWorkflowCollectionAlignmentAnchored specifies the alignment for
 * collecting the pending line items into an invoice.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingWorkflowCollectionAlignmentAnchored")
model V1BillingWorkflowCollectionAlignmentAnchored {
  /**
   * The type of alignment.
   */
  type: "anchored";

  /**
   * The recurring period for the alignment.
   */
  recurring_period: V1RecurringPeriodV2;
}

/**
 * BillingWorkflowCollectionAlignmentSubscription specifies the alignment for
 * collecting the pending line items into an invoice.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingWorkflowCollectionAlignmentSubscription")
model V1BillingWorkflowCollectionAlignmentSubscription {
  /**
   * The type of alignment.
   */
  type: "subscription";
}

/**
 * Workflow collection specifies how to collect the pending line items for an
 * invoice
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingWorkflowCollectionSettings")
model V1BillingWorkflowCollectionSettings {
  /**
   * The alignment for collecting the pending line items into an invoice.
   */
  alignment?: V1BillingWorkflowCollectionAlignment;

  /**
   * This grace period can be used to delay the collection of the pending line items
   * specified in alignment.
   *
   * This is useful, in case of multiple subscriptions having slightly different
   * billing periods.
   */
  @format("ISO8601") interval?: string;
}

/**
 * BillingWorkflowInvoicingSettings represents the invoice settings for a billing
 * workflow
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@summary("Workflow invoice settings")
@friendlyName("BillingNotificationEventV1BillingWorkflowInvoicingSettings")
model V1BillingWorkflowInvoicingSettings {
  /**
   * Whether to automatically issue the invoice after the draftPeriod has passed.
   */
  auto_advance?: boolean;

  /**
   * The period for the invoice to be kept in draft status for manual reviews.
   */
  @format("ISO8601") draft_period?: string;

  /**
   * The period after which the invoice is due. With some payment solutions it's only
   * applicable for manual collection method.
   */
  @format("ISO8601") due_after?: string;

  /**
   * Should progressive billing be allowed for this workflow?
   */
  progressive_billing?: boolean;

  /**
   * Controls how subscription-ending shortened service periods are billed.
   */
  subscription_end_proration_mode?: V1BillingWorkflowInvoicingSubscriptionEndProrationMode;

  /**
   * Default tax configuration to apply to the invoices.
   *
   * Setting a tax code (`stripe.code` / `taxCodeId`) on a profile's default tax
   * config is deprecated and can no longer be added or changed: the organization
   * default tax code is used instead. Existing tax-code values may still be removed,
   * and `behavior` remains fully supported.
   */
  default_tax_config?: V1TaxConfig;
}

/**
 * Billing workflow subscription end proration mode.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BillingWorkflowInvoicingSubscriptionEndProrationMode")
enum V1BillingWorkflowInvoicingSubscriptionEndProrationMode {
  bill_full_period,
  bill_actual_period,
}

/**
 * BillingWorkflowPaymentSettings represents the payment settings for a billing
 * workflow
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@summary("Workflow payment settings")
@friendlyName("BillingNotificationEventV1BillingWorkflowPaymentSettings")
model V1BillingWorkflowPaymentSettings {
  /**
   * The payment method for the invoice.
   */
  collection_method?: V1CollectionMethod;
}

/**
 * BillingWorkflowTaxSettings represents the tax settings for a billing workflow
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@summary("Workflow tax settings")
@friendlyName("BillingNotificationEventV1BillingWorkflowTaxSettings")
model V1BillingWorkflowTaxSettings {
  /**
   * Enable automatic tax calculation when tax is supported by the app. For example,
   * with Stripe Invoicing when enabled, tax is calculated via Stripe Tax.
   */
  enabled?: boolean;

  /**
   * Enforce tax calculation when tax is supported by the app. When enabled,
   * OpenMeter will not allow to create an invoice without tax calculation.
   * Enforcement is different per apps, for example, Stripe app requires customer to
   * have a tax location when starting a paid subscription.
   */
  enforced?: boolean;
}

/**
 * CollectionMethod specifies how the invoice should be collected (automatic vs
 * manual)
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@summary("Collection method")
@friendlyName("BillingNotificationEventV1CollectionMethod")
enum V1CollectionMethod {
  charge_automatically,
  send_invoice,
}

/**
 * [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country
 * code. Custom two-letter country codes are also supported for convenience.
 */
@minLength(2)
@maxLength(2)
@pattern("^[A-Z]{2}$")
@friendlyName("BillingNotificationEventV1CountryCode")
scalar V1CountryCode extends string;

/**
 * Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html)
 * currency code. Custom three-letter currency codes are also supported for
 * convenience.
 */
@minLength(3)
@maxLength(3)
@pattern("^[A-Z]{3}$")
@friendlyName("BillingNotificationEventV1CurrencyCode")
scalar V1CurrencyCode extends string;

/**
 * Custom Invoicing app can be used for interface with any invoicing or payment
 * system.
 *
 * This app provides ways to manipulate invoices and payments, however the
 * integration must rely on Notifications API to get notified about invoice
 * changes.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1CustomInvoicingApp")
model V1CustomInvoicingApp {
  /**
   * A unique identifier for the resource.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  @summary("ID")
  id: string;

  /**
   * Human-readable name for the resource. Between 1 and 256 characters.
   */
  @minLength(1)
  @maxLength(256)
  @summary("Display name")
  name: string;

  /**
   * Optional description of the resource. Maximum 1024 characters.
   */
  @maxLength(1024) @summary("Description") description?: string;

  /**
   * Additional metadata for the resource.
   */
  @summary("Metadata") metadata?: V1Metadata | null;

  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * The marketplace listing that this installed app is based on.
   */
  @visibility(Lifecycle.Read) listing: V1MarketplaceListing;

  /**
   * Status of the app connection.
   */
  @visibility(Lifecycle.Read) status: V1AppStatus;

  /**
   * The app's type is CustomInvoicing.
   */
  type: "custom_invoicing";

  /**
   * Enable draft.sync hook.
   *
   * If the hook is not enabled, the invoice will be progressed to the next state
   * automatically.
   */
  enable_draft_sync_hook: boolean;

  /**
   * Enable issuing.sync hook.
   *
   * If the hook is not enabled, the invoice will be progressed to the next state
   * automatically.
   */
  enable_issuing_sync_hook: boolean;
}

/**
 * Custom invoicing tax config.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1CustomInvoicingTaxConfig")
model V1CustomInvoicingTaxConfig {
  /**
   * Tax code.
   *
   * The tax code should be interpreted by the custom invoicing provider.
   */
  @summary("Tax code") code: string;
}

/**
 * A customer object.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1Customer")
model V1Customer {
  /**
   * A unique identifier for the resource.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  @summary("ID")
  id: string;

  /**
   * Human-readable name for the resource. Between 1 and 256 characters.
   */
  @minLength(1)
  @maxLength(256)
  @summary("Display name")
  name: string;

  /**
   * Optional description of the resource. Maximum 1024 characters.
   */
  @maxLength(1024) @summary("Description") description?: string;

  /**
   * Additional metadata for the resource.
   */
  @summary("Metadata") metadata?: V1Metadata | null;

  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * An optional unique key of the customer. Either key or
   * usageAttribution.subjectKeys must be provided. Useful to reference the customer
   * in external systems. For example, your database ID.
   */
  @minLength(1)
  @maxLength(256)
  @summary("Key")
  key?: string;

  /**
   * Mapping to attribute metered usage to the customer Either key or
   * usageAttribution.subjectKeys must be provided.
   */
  @summary("Usage Attribution") usage_attribution?: V1CustomerUsageAttribution;

  /**
   * The primary email address of the customer.
   */
  @summary("Primary Email") primary_email?: string;

  /**
   * Currency of the customer. Used for billing, tax and invoicing.
   */
  @summary("Currency") currency?: V1CurrencyCode;

  /**
   * The billing address of the customer. Used for tax and invoicing.
   */
  @summary("Billing Address") billing_address?: V1Address;

  /**
   * The ID of the Subscription if the customer has one.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  @summary("Current Subscription ID")
  current_subscription_id?: string;

  /**
   * The subscriptions of the customer. Only with the `subscriptions` expand option.
   */
  @visibility(Lifecycle.Read)
  @summary("Subscriptions")
  subscriptions?: V1Subscription[];

  /**
   * Set of key-value pairs managed by the system. Cannot be modified by user.
   */
  @visibility(Lifecycle.Read)
  @summary("Annotations")
  annotations?: V1Annotations;
}

/**
 * Mapping to attribute metered usage to the customer. One customer can have zero
 * or more subjects, but one subject can only belong to one customer.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1CustomerUsageAttribution")
model V1CustomerUsageAttribution {
  /**
   * The subjects that are attributed to the customer. Can be empty when no subjects
   * are associated with the customer.
   */
  @minItems(0) @summary("SubjectKeys") subject_keys: string[];
}

/**
 * The reason for the discount is a maximum spend.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1DiscountReasonMaximumSpend")
model V1DiscountReasonMaximumSpend {
  /**
   * type.
   */
  type: "maximum_spend";
}

/**
 * The reason for the discount is a ratecard percentage.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1DiscountReasonRatecardPercentage")
model V1DiscountReasonRatecardPercentage {
  /**
   * type.
   */
  type: "ratecard_percentage";

  /**
   * The percentage of the discount.
   */
  @minValue(0)
  @maxValue(100)
  @summary("Percentage")
  percentage: V1Percentage;

  /**
   * Correlation ID for the discount.
   *
   * This is used to link discounts across different invoices (progressive billing
   * use case).
   *
   * If not provided, the invoicing engine will auto-generate one. When editing an
   * invoice line, please make sure to keep the same correlation ID of the discount
   * or in progressive billing setups the discount amounts might be incorrect.
   */
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  correlation_id?: string;
}

/**
 * The reason for the discount is a ratecard usage.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1DiscountReasonRatecardUsage")
model V1DiscountReasonRatecardUsage {
  /**
   * type.
   */
  type: "ratecard_usage";

  /**
   * The quantity of the usage discount.
   *
   * Must be positive.
   */
  @summary("Usage") quantity: V1Numeric;

  /**
   * Correlation ID for the discount.
   *
   * This is used to link discounts across different invoices (progressive billing
   * use case).
   *
   * If not provided, the invoicing engine will auto-generate one. When editing an
   * invoice line, please make sure to keep the same correlation ID of the discount
   * or in progressive billing setups the discount amounts might be incorrect.
   */
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  correlation_id?: string;
}

/**
 * Dynamic price with spend commitments.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1DynamicPriceWithCommitments")
model V1DynamicPriceWithCommitments {
  /**
   * The type of the price.
   */
  type: "dynamic";

  /**
   * The multiplier to apply to the base price to get the dynamic price.
   *
   * Examples:
   *
   * - 0.0: the price is zero
   * - 0.5: the price is 50% of the base price
   * - 1.0: the price is the same as the base price
   * - 1.5: the price is 150% of the base price
   */
  @summary("The multiplier to apply to the base price to get the dynamic price")
  multiplier?: V1Numeric;

  /**
   * The customer is committed to spend at least the amount.
   */
  @summary("Minimum amount") minimum_amount?: V1Numeric;

  /**
   * The customer is limited to spend at most the amount.
   */
  @summary("Maximum amount") maximum_amount?: V1Numeric;
}

/**
 * Metered entitlements are useful for many different use cases, from setting up
 * usage based access to implementing complex credit systems. Access is determined
 * based on feature usage using a balance calculation (the "usage allowance"
 * provided by the issued grants is "burnt down" by the usage).
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "Copied v1 field layout."
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
#deprecated "deprecated"
@friendlyName("BillingNotificationEventV1EntitlementMetered")
model V1EntitlementMetered {
  /**
   * type.
   */
  type: "metered";

  /**
   * If softLimit=true the subject can use the feature even if the entitlement is
   * exhausted, hasAccess will always be true.
   */
  @summary("Soft limit") is_soft_limit?: boolean;

  /**
   * Deprecated, ignored by the backend. Please use isSoftLimit instead; this field
   * will be removed in the future.
   */
  #deprecated "deprecated"
  is_unlimited?: boolean;

  /**
   * You can grant usage automatically alongside the entitlement, the example
   * scenario would be creating a starting balance. If an amount is specified here, a
   * grant will be created alongside the entitlement with the specified amount. That
   * grant will have it's rollover settings configured in a way that after each reset
   * operation, the balance will return the original amount specified here. Manually
   * creating such a grant would mean having the "amount", "minRolloverAmount", and
   * "maxRolloverAmount" fields all be the same.
   */
  @minValue(0) @summary("Initial grant amount") issue_after_reset?: float64;

  /**
   * Defines the grant priority for the default grant.
   */
  @minValue(1)
  @maxValue(255)
  @summary("Issue grant after reset priority")
  issue_after_reset_priority?: uint8;

  /**
   * If true, the overage is preserved at reset. If false, the usage is reset to 0.
   */
  @summary("Preserve overage at reset")
  preserve_overage_at_reset?: boolean;

  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * Additional metadata for the feature.
   */
  metadata?: V1Metadata;

  /**
   * The cadence start of the resource.
   */
  active_from: utcDateTime;

  /**
   * The cadence end of the resource.
   */
  active_to?: utcDateTime;

  /**
   * The annotations of the entitlement.
   */
  @visibility(Lifecycle.Read) annotations?: V1Annotations;

  /**
   * Readonly unique ULID identifier.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  id: string;

  /**
   * The identifier key unique to the subject. NOTE: Subjects are being deprecated,
   * please use the new customer APIs.
   */
  subject_key: string;

  /**
   * The feature the subject is entitled to use.
   */
  @minLength(1)
  @maxLength(64)
  @pattern("^[a-z0-9]+(?:_[a-z0-9]+)*$")
  feature_key: string;

  /**
   * The feature the subject is entitled to use.
   */
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$") feature_id: string;

  /**
   * The time the last reset happened.
   */
  @visibility(Lifecycle.Read) last_reset: utcDateTime;

  /**
   * The current usage period.
   */
  @visibility(Lifecycle.Read) current_usage_period: V1Period;

  /**
   * The time from which usage is measured. If not specified on creation, defaults to
   * entitlement creation time.
   */
  @visibility(Lifecycle.Read) measure_usage_from: utcDateTime;

  /**
   * THe usage period of the entitlement.
   */
  #suppress "deprecated" "The v1 payload still carries this deprecated v1 model."
  @visibility(Lifecycle.Read) usage_period: V1RecurringPeriod;
}

/**
 * Entitlements are the core of OpenMeter access management. They define access to
 * features for subjects. Entitlements can be metered, boolean, or static.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1EntitlementValue")
model V1EntitlementValue {
  /**
   * Whether the subject has access to the feature. Shared accross all entitlement
   * types.
   */
  @visibility(Lifecycle.Read) has_access: boolean;

  /**
   * Only available for metered entitlements. Metered entitlements are built around a
   * balance calculation where feature usage is deducted from the issued grants.
   * Balance represents the remaining balance of the entitlement, it's value never
   * turns negative.
   */
  @visibility(Lifecycle.Read) balance?: float64;

  /**
   * Only available for metered entitlements. Returns the total feature usage in the
   * current period.
   */
  @visibility(Lifecycle.Read) usage?: float64;

  /**
   * Only available for metered entitlements. Overage represents the usage that
   * wasn't covered by grants, e.g. if the subject had a total feature usage of 100
   * in the period but they were only granted 80, there would be 20 overage.
   */
  @visibility(Lifecycle.Read) overage?: float64;

  /**
   * Only available for metered entitlements. The summed amount of all grant active
   * at query time PLUS the used amount of since inactive grants.
   */
  @visibility(Lifecycle.Read) total_available_grant_amount?: float64;

  /**
   * Only available for static entitlements. The JSON parsable config of the
   * entitlement.
   */
  @visibility(Lifecycle.Read) config?: string;
}

/**
 * Represents a feature that can be enabled or disabled for a plan. Used both for
 * product catalog and entitlements.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "Copied v1 field layout."
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1Feature")
model V1Feature {
  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * Timestamp of when the resource was archived.
   */
  @visibility(Lifecycle.Read)
  @summary("Archival Time")
  archived_at?: utcDateTime;

  /**
   * A key is a unique string that is used to identify a resource.
   */
  @minLength(1)
  @maxLength(64)
  @pattern("^[a-z0-9]+(?:_[a-z0-9]+)*$")
  @summary("The unique key of the feature")
  key: string;

  /**
   * The human-readable name of the feature.
   */
  @summary("The human-readable name of the feature") name: string;

  /**
   * Optional metadata.
   */
  @summary("Optional metadata") metadata?: V1Metadata;

  /**
   * A key is a unique string that is used to identify a resource.
   */
  @minLength(1)
  @summary("Meter slug")
  meter_slug?: string;

  /**
   * Optional meter group by filters. Useful if the meter scope is broader than what
   * feature tracks. Example scenario would be a meter tracking all token use with
   * groupBy fields for the model, then the feature could filter for model=gpt-4.
   *
   * ⚠️ **Deprecated**: Use advancedMeterGroupByFilters instead
   */
  #deprecated "deprecated"
  @summary("Meter group by filters") meter_group_by_filters?: Record<string>;

  /**
   * Optional advanced meter group by filters. You can use this to filter for values
   * of the meter groupBy fields.
   */
  @summary("Advanced meter group by filters")
  advanced_meter_group_by_filters?: Record<V1FilterString>;

  /**
   * Optional per-unit cost configuration. Use "manual" for a fixed per-unit cost, or
   * "llm" to look up cost from the LLM cost database based on meter group-by
   * properties.
   */
  @summary("Unit cost") unit_cost?: V1FeatureUnitCost;

  /**
   * Readonly unique ULID identifier.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  id: string;
}

/**
 * LLM cost lookup configuration. Maps meter group-by dimensions to LLM cost
 * database fields.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "Copied v1 field layout."
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1FeatureLLMUnitCost")
model V1FeatureLLMUnitCost {
  /**
   * type.
   */
  type: "llm";

  /**
   * Meter group-by property that holds the LLM provider. Use this when the meter has
   * a group-by dimension for provider. Mutually exclusive with `provider`.
   */
  @summary("Provider property") provider_property?: string;

  /**
   * Static LLM provider value (e.g., "openai", "anthropic"). Use this when the
   * feature tracks a single provider. Mutually exclusive with `providerProperty`.
   */
  @summary("Provider") provider?: string;

  /**
   * Meter group-by property that holds the model ID. Use this when the meter has a
   * group-by dimension for model. Mutually exclusive with `model`.
   */
  @summary("Model property") model_property?: string;

  /**
   * Static model ID value (e.g., "gpt-4", "claude-3-5-sonnet"). Use this when the
   * feature tracks a single model. Mutually exclusive with `modelProperty`.
   */
  @summary("Model") `model`?: string;

  /**
   * Meter group-by property that holds the token type. Use this when the meter has a
   * group-by dimension for token type. Mutually exclusive with `tokenType`.
   */
  @summary("Token type property") token_type_property?: string;

  /**
   * Static token type value. Use this when the feature tracks a single token type
   * (e.g., only input tokens). Expected values: input, output, cache_read,
   * reasoning, cache_write, request, response. `request` is an alias for `input`,
   * `response` is an alias for `output`. Mutually exclusive with
   * `tokenTypeProperty`.
   */
  @summary("Token type") token_type?: string;

  /**
   * Resolved per-token pricing from the LLM cost database. Only populated in
   * responses when the feature's meter group-by filters specify exact provider and
   * model values.
   */
  @visibility(Lifecycle.Read)
  @summary("Resolved pricing")
  pricing?: V1FeatureLLMUnitCostPricing;
}

/**
 * Resolved per-token pricing from the LLM cost database.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "Copied v1 field layout."
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1FeatureLLMUnitCostPricing")
model V1FeatureLLMUnitCostPricing {
  /**
   * Cost per input token in USD.
   */
  @summary("Input per token") input_per_token: V1Numeric;

  /**
   * Cost per output token in USD.
   */
  @summary("Output per token") output_per_token: V1Numeric;

  /**
   * Cost per cache read token in USD.
   */
  @summary("Cache read per token") cache_read_per_token?: V1Numeric;

  /**
   * Cost per reasoning token in USD.
   */
  @summary("Reasoning per token") reasoning_per_token?: V1Numeric;

  /**
   * Cost per cache write token in USD.
   */
  @summary("Cache write per token") cache_write_per_token?: V1Numeric;
}

/**
 * A fixed per-unit cost amount.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1FeatureManualUnitCost")
model V1FeatureManualUnitCost {
  /**
   * type.
   */
  type: "manual";

  /**
   * Fixed per-unit cost amount in USD.
   */
  amount: V1Numeric;
}

/**
 * Per-unit cost configuration for a feature. Either a fixed manual amount or a
 * dynamic LLM cost lookup.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
@oneOf
@friendlyName("BillingNotificationEventV1FeatureUnitCost")
union V1FeatureUnitCost {
  /**
   * manual.
   */
  manual: V1FeatureManualUnitCost,

  /**
   * llm.
   */
  llm: V1FeatureLLMUnitCost,
}

/**
 * A filter for a string field.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1FilterString")
model V1FilterString {
  /**
   * The field must be equal to the provided value.
   */
  @encodedName("application/json", "$eq")
  @extension("x-omitempty", true)
  eq?: string | null;

  /**
   * The field must not be equal to the provided value.
   */
  @encodedName("application/json", "$ne")
  @extension("x-omitempty", true)
  ne?: string | null;

  /**
   * The field must be in the provided list of values.
   */
  @encodedName("application/json", "$in")
  @extension("x-omitempty", true)
  in?: string[] | null;

  /**
   * The field must not be in the provided list of values.
   */
  @encodedName("application/json", "$nin")
  @extension("x-omitempty", true)
  nin?: string[] | null;

  /**
   * The field must match the provided value.
   */
  @encodedName("application/json", "$like")
  @extension("x-omitempty", true)
  like?: string | null;

  /**
   * The field must not match the provided value.
   */
  @encodedName("application/json", "$nlike")
  @extension("x-omitempty", true)
  nlike?: string | null;

  /**
   * The field must match the provided value, ignoring case.
   */
  @encodedName("application/json", "$ilike")
  @extension("x-omitempty", true)
  ilike?: string | null;

  /**
   * The field must not match the provided value, ignoring case.
   */
  @encodedName("application/json", "$nilike")
  @extension("x-omitempty", true)
  nilike?: string | null;

  /**
   * The field must be greater than the provided value.
   */
  @encodedName("application/json", "$gt")
  @extension("x-omitempty", true)
  gt?: string | null;

  /**
   * The field must be greater than or equal to the provided value.
   */
  @encodedName("application/json", "$gte")
  @extension("x-omitempty", true)
  gte?: string | null;

  /**
   * The field must be less than the provided value.
   */
  @encodedName("application/json", "$lt")
  @extension("x-omitempty", true)
  lt?: string | null;

  /**
   * The field must be less than or equal to the provided value.
   */
  @encodedName("application/json", "$lte")
  @extension("x-omitempty", true)
  lte?: string | null;

  /**
   * Provide a list of filters to be combined with a logical AND.
   */
  @encodedName("application/json", "$and")
  @extension("x-omitempty", true)
  and?: V1FilterString[] | null;

  /**
   * Provide a list of filters to be combined with a logical OR.
   */
  @encodedName("application/json", "$or")
  @extension("x-omitempty", true)
  or?: V1FilterString[] | null;
}

/**
 * Flat price.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1FlatPrice")
model V1FlatPrice {
  /**
   * The type of the price.
   */
  type: "flat";

  /**
   * The amount of the flat price.
   */
  amount: V1Numeric;
}

/**
 * Flat price with payment term.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1FlatPriceWithPaymentTerm")
model V1FlatPriceWithPaymentTerm {
  /**
   * The type of the price.
   */
  type: "flat";

  /**
   * The amount of the flat price.
   */
  amount: V1Numeric;

  /**
   * The payment term of the flat price. Defaults to in advance.
   */
  payment_term?: V1PricePaymentTerm;
}

/**
 * IDResource is a resouce with an ID.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1IDResource")
model V1IDResource {
  /**
   * A unique identifier for the resource.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  @summary("ID")
  id: string;
}

/**
 * Install method of the application.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InstallMethod")
enum V1InstallMethod {
  with_oauth2,
  with_api_key,
  no_credentials_required,
}

/**
 * Invoice represents an invoice in the system.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1Invoice")
model V1Invoice {
  /**
   * A unique identifier for the resource.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  @summary("ID")
  id: string;

  /**
   * Optional description of the resource. Maximum 1024 characters.
   */
  @maxLength(1024) @summary("Description") description?: string;

  /**
   * Additional metadata for the resource.
   */
  @summary("Metadata") metadata?: V1Metadata | null;

  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * Type of the invoice.
   *
   * The type of invoice determines the purpose of the invoice and how it should be
   * handled.
   *
   * Supported types:
   *
   * - standard: A regular commercial invoice document between a supplier and
   *   customer.
   * - credit_note: Reflects a refund either partial or complete of the preceding
   *   document. A credit note effectively _extends_ the previous document.
   */
  @visibility(Lifecycle.Read) type: V1InvoiceType;

  /**
   * The taxable entity supplying the goods or services.
   */
  supplier: V1BillingParty;

  /**
   * Legal entity receiving the goods or services.
   */
  customer: V1BillingInvoiceCustomerExtendedDetails;

  /**
   * Number specifies the human readable key used to reference this Invoice.
   *
   * The invoice number can change in the draft phases, as we are allocating
   * temporary draft invoice numbers, but it's final as soon as the invoice gets
   * finalized (issued state).
   *
   * Please note that the number is (depending on the upstream settings) either
   * unique for the whole organization or unique for the customer, or in multi
   * (stripe) account setups unique for the account.
   */
  @visibility(Lifecycle.Read) number: V1InvoiceNumber;

  /**
   * Currency for all invoice line items.
   *
   * Multi currency invoices are not supported yet.
   */
  currency: V1CurrencyCode;

  /**
   * Key information regarding previous invoices and potentially details as to why
   * they were corrected.
   */
  @visibility(Lifecycle.Read) preceding?: V1InvoiceDocumentRef[];

  /**
   * Summary of all the invoice totals, including taxes (calculated).
   */
  @visibility(Lifecycle.Read) totals: V1InvoiceTotals;

  /**
   * The status of the invoice.
   *
   * This field only conatins a simplified status, for more detailed information use
   * the statusDetails field.
   */
  @visibility(Lifecycle.Read) status: V1InvoiceStatus;

  /**
   * The details of the current invoice status.
   */
  @visibility(Lifecycle.Read) status_details: V1InvoiceStatusDetails;

  /**
   * The time the invoice was issued.
   *
   * Depending on the status of the invoice this can mean multiple things:
   *
   * - draft, gathering: The time the invoice will be issued based on the workflow
   *   settings.
   * - issued: The time the invoice was issued.
   */
  @visibility(Lifecycle.Read) issued_at?: utcDateTime;

  /**
   * The time until the invoice is in draft status.
   *
   * On draft invoice creation it is calculated from the workflow settings.
   *
   * If manual approval is required, the draftUntil time is set.
   */
  draft_until?: utcDateTime;

  /**
   * The time when the quantity snapshots on the invoice lines were taken.
   */
  @visibility(Lifecycle.Read) quantity_snapshoted_at?: utcDateTime;

  /**
   * The time when the invoice will be/has been collected.
   */
  @visibility(Lifecycle.Read) collection_at?: utcDateTime;

  /**
   * Due time of the fulfillment of the invoice (if available).
   */
  @visibility(Lifecycle.Read) due_at?: utcDateTime;

  /**
   * The period the invoice covers. If the invoice has no line items, it's not set.
   */
  period?: V1Period;

  /**
   * The time the invoice was voided.
   *
   * If the invoice was voided, this field will be set to the time the invoice was
   * voided.
   */
  @visibility(Lifecycle.Read) voided_at?: utcDateTime;

  /**
   * The time the invoice was sent to customer.
   */
  @visibility(Lifecycle.Read) sent_to_customer_at?: utcDateTime;

  /**
   * The workflow associated with the invoice.
   *
   * It is always a snapshot of the workflow settings at the time of invoice
   * creation. The field is optional as it should be explicitly requested with expand
   * options.
   */
  workflow: V1InvoiceWorkflowSettings;

  /**
   * List of invoice lines representing each of the items sold to the customer.
   */
  lines?: V1InvoiceLine[];

  /**
   * Information on when, how, and to whom the invoice should be paid.
   */
  @visibility(Lifecycle.Read) payment?: V1InvoicePaymentTerms;

  /**
   * Validation issues reported by the invoice workflow.
   */
  @visibility(Lifecycle.Read) validation_issues?: V1ValidationIssue[];

  /**
   * External IDs of the invoice in other apps such as Stripe.
   */
  @visibility(Lifecycle.Read) external_ids?: V1InvoiceAppExternalIds;
}

/**
 * InvoiceAppExternalIds contains the external IDs of the invoice in other apps
 * such as Stripe.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceAppExternalIds")
model V1InvoiceAppExternalIds {
  /**
   * The external ID of the invoice in the invoicing app if available.
   */
  @visibility(Lifecycle.Read) invoicing?: string;

  /**
   * The external ID of the invoice in the tax app if available.
   */
  @visibility(Lifecycle.Read) tax?: string;

  /**
   * The external ID of the invoice in the payment app if available.
   */
  @visibility(Lifecycle.Read) payment?: string;
}

/**
 * InvoiceAvailableActionInvoiceDetails represents the details of the invoice
 * action for non-gathering invoices.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceAvailableActionDetails")
model V1InvoiceAvailableActionDetails {
  /**
   * The state the invoice will reach if the action is activated and all intermediate
   * steps are successful.
   *
   * For example advancing a draft_created invoice will result in a
   * draft_manual_approval_needed invoice.
   */
  @visibility(Lifecycle.Read) resulting_state: string;
}

/**
 * InvoiceAvailableActionInvoiceDetails represents the details of the invoice
 * action for gathering invoices.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceAvailableActionInvoiceDetails")
model V1InvoiceAvailableActionInvoiceDetails {}

/**
 * InvoiceAvailableActions represents the actions that can be performed on the
 * invoice.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceAvailableActions")
model V1InvoiceAvailableActions {
  /**
   * Advance the invoice to the next status.
   */
  @visibility(Lifecycle.Read) advance?: V1InvoiceAvailableActionDetails;

  /**
   * Approve an invoice that requires manual approval.
   */
  @visibility(Lifecycle.Read) approve?: V1InvoiceAvailableActionDetails;

  /**
   * Delete the invoice (only non-issued invoices can be deleted).
   */
  @visibility(Lifecycle.Read) delete?: V1InvoiceAvailableActionDetails;

  /**
   * Retry an invoice issuing step that failed.
   */
  @visibility(Lifecycle.Read) retry?: V1InvoiceAvailableActionDetails;

  /**
   * Snapshot quantities for usage based line items.
   */
  @visibility(Lifecycle.Read)
  snapshot_quantities?: V1InvoiceAvailableActionDetails;

  /**
   * Void an already issued invoice.
   */
  @visibility(Lifecycle.Read) `void`?: V1InvoiceAvailableActionDetails;

  /**
   * Invoice a gathering invoice
   */
  @visibility(Lifecycle.Read) invoice?: V1InvoiceAvailableActionInvoiceDetails;
}

/**
 * InvoiceDetailedLine represents a line item that is sold to the customer as a
 * manually added fee.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceDetailedLine")
model V1InvoiceDetailedLine {
  /**
   * Human-readable name for the resource. Between 1 and 256 characters.
   */
  @minLength(1)
  @maxLength(256)
  @summary("Display name")
  name: string;

  /**
   * Optional description of the resource. Maximum 1024 characters.
   */
  @maxLength(1024) @summary("Description") description?: string;

  /**
   * Additional metadata for the resource.
   */
  @summary("Metadata") metadata?: V1Metadata | null;

  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * ID of the line.
   */
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$") id: string;

  /**
   * managedBy specifies if the line is manually added via the api or managed by
   * OpenMeter.
   */
  @visibility(Lifecycle.Read) managed_by: V1InvoiceLineManagedBy;

  /**
   * Status of the line.
   *
   * External calls always create valid lines, other line types are managed by the
   * billing engine of OpenMeter.
   */
  @visibility(Lifecycle.Read) status: V1InvoiceLineStatus;

  /**
   * Discounts detailes applied to this line.
   *
   * New discounts can be added via the invoice's discounts API, to facilitate
   * discounts that are affecting multiple lines.
   */
  @visibility(Lifecycle.Read) discounts?: V1InvoiceLineDiscounts;

  /**
   * Credit allocations applied to this line.
   *
   * Credits are deducted from the line total before taxes are applied.
   */
  @visibility(Lifecycle.Read)
  credit_allocations?: V1InvoiceLineCreditAllocation[];

  /**
   * The invoice this item belongs to.
   */
  invoice?: V1InvoiceReference;

  /**
   * The currency of this line.
   */
  currency: V1CurrencyCode;

  /**
   * Taxes applied to the invoice totals.
   */
  @visibility(Lifecycle.Read) taxes?: V1InvoiceLineTaxItem[];

  /**
   * Tax config specify the tax configuration for this line.
   */
  #deprecated "deprecated"
  tax_config?: V1TaxConfig;

  /**
   * Totals for this line.
   */
  @visibility(Lifecycle.Read) totals: V1InvoiceTotals;

  /**
   * Period of the line item applies to for revenue recognition pruposes.
   *
   * Billing always treats periods as start being inclusive and end being exclusive.
   */
  period: V1Period;

  /**
   * External IDs of the invoice in other apps such as Stripe.
   */
  @visibility(Lifecycle.Read) external_ids?: V1InvoiceLineAppExternalIds;

  /**
   * Subscription are the references to the subscritpions that this line is related
   * to.
   */
  @visibility(Lifecycle.Read) subscription?: V1InvoiceLineSubscriptionReference;

  /**
   * The time this line item should be invoiced.
   */
  #deprecated "deprecated"
  invoice_at: utcDateTime;

  /**
   * Type of the line.
   */
  #deprecated "deprecated"
  @visibility(Lifecycle.Read) type: "flat_fee";

  /**
   * Price of the item being sold.
   */
  #deprecated "deprecated"
  per_unit_amount?: V1Numeric;

  /**
   * Payment term of the line.
   */
  #deprecated "deprecated"
  payment_term?: V1PricePaymentTerm;

  /**
   * Quantity of the item being sold.
   */
  #deprecated "deprecated"
  quantity?: V1Numeric;

  /**
   * The rate card that is used for this line.
   */
  rate_card?: V1InvoiceDetailedLineRateCard;

  /**
   * Category of the flat fee.
   */
  @visibility(Lifecycle.Read)
  category?: V1InvoiceDetailedLineCostCategory;
}

/**
 * InvoiceDetailedLineCostCategory determines if the flat fee is a regular fee due
 * to use due to a commitment.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceDetailedLineCostCategory")
enum V1InvoiceDetailedLineCostCategory {
  regular,
  commitment,
}

/**
 * InvoiceDetailedLineRateCard represents the rate card (intent) for a flat fee
 * line.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
#suppress "@openmeter/api-spec-aip/no-nullable" "Legacy v1 wire shape: the field is required and nullable."
@friendlyName("BillingNotificationEventV1InvoiceDetailedLineRateCard")
model V1InvoiceDetailedLineRateCard {
  /**
   * The tax config of the rate card. When undefined, the tax config of the feature
   * or the default tax config of the plan is used.
   */
  @summary("Tax config") tax_config?: V1TaxConfig;

  /**
   * The price of the rate card. When null, the feature or service is free.
   */
  @summary("Price") price: V1FlatPriceWithPaymentTerm | null;

  /**
   * Quantity of the item being sold.
   *
   * Default: 1
   */
  quantity?: V1Numeric;

  /**
   * The discounts that are applied to the line.
   */
  discounts?: V1BillingDiscounts;
}

/**
 * InvoiceDocumentRef is used to describe a reference to an existing document
 * (invoice).
 *
 * The v1 schema describes credit note references with issued_at, number and url,
 * but the v1 API only ever sends the generic fields below.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceDocumentRef")
model V1InvoiceDocumentRef {
  ...V1InvoiceGenericDocumentRef;
}

/**
 * InvoiceDocumentRefType defines the type of document that is being referenced.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceDocumentRefType")
enum V1InvoiceDocumentRefType {
  credit_note_original_invoice,
}

/**
 * Omitted fields: period: Tax period in which the referred document had an effect
 * required by some tax regimes and formats. stamps: Seals of approval from other
 * organisations that may need to be listed. ext: Extensions for additional codes
 * that may be required.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@summary("InvoiceGenericDocumentRef is used to describe an existing document or a specific part of it's contents.")
@friendlyName("BillingNotificationEventV1InvoiceGenericDocumentRef")
model V1InvoiceGenericDocumentRef {
  /**
   * Type of the document referenced.
   */
  @visibility(Lifecycle.Read) type: V1InvoiceDocumentRefType;

  /**
   * Human readable description on why this reference is here or needs to be used.
   */
  @visibility(Lifecycle.Read) reason?: string;

  /**
   * Additional details about the document.
   */
  @visibility(Lifecycle.Read) description?: string;
}

/**
 * InvoiceUsageBasedLine represents a line item that is sold to the customer based
 * on usage.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "Copied v1 field layout."
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceLine")
model V1InvoiceLine {
  /**
   * Human-readable name for the resource. Between 1 and 256 characters.
   */
  @minLength(1)
  @maxLength(256)
  @summary("Display name")
  name: string;

  /**
   * Optional description of the resource. Maximum 1024 characters.
   */
  @maxLength(1024) @summary("Description") description?: string;

  /**
   * Additional metadata for the resource.
   */
  @summary("Metadata") metadata?: V1Metadata | null;

  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * ID of the line.
   */
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$") id: string;

  /**
   * managedBy specifies if the line is manually added via the api or managed by
   * OpenMeter.
   */
  @visibility(Lifecycle.Read) managed_by: V1InvoiceLineManagedBy;

  /**
   * Status of the line.
   *
   * External calls always create valid lines, other line types are managed by the
   * billing engine of OpenMeter.
   */
  @visibility(Lifecycle.Read) status: V1InvoiceLineStatus;

  /**
   * Discounts detailes applied to this line.
   *
   * New discounts can be added via the invoice's discounts API, to facilitate
   * discounts that are affecting multiple lines.
   */
  @visibility(Lifecycle.Read) discounts?: V1InvoiceLineDiscounts;

  /**
   * Credit allocations applied to this line.
   *
   * Credits are deducted from the line total before taxes are applied.
   */
  @visibility(Lifecycle.Read)
  credit_allocations?: V1InvoiceLineCreditAllocation[];

  /**
   * The invoice this item belongs to.
   */
  invoice?: V1InvoiceReference;

  /**
   * The currency of this line.
   */
  currency: V1CurrencyCode;

  /**
   * Taxes applied to the invoice totals.
   */
  @visibility(Lifecycle.Read) taxes?: V1InvoiceLineTaxItem[];

  /**
   * Tax config specify the tax configuration for this line.
   */
  #deprecated "deprecated"
  tax_config?: V1TaxConfig;

  /**
   * Totals for this line.
   */
  @visibility(Lifecycle.Read) totals: V1InvoiceTotals;

  /**
   * Period of the line item applies to for revenue recognition pruposes.
   *
   * Billing always treats periods as start being inclusive and end being exclusive.
   */
  period: V1Period;

  /**
   * The time this line item should be invoiced.
   */
  invoice_at: utcDateTime;

  /**
   * External IDs of the invoice in other apps such as Stripe.
   */
  @visibility(Lifecycle.Read) external_ids?: V1InvoiceLineAppExternalIds;

  /**
   * Subscription are the references to the subscritpions that this line is related
   * to.
   */
  @visibility(Lifecycle.Read) subscription?: V1InvoiceLineSubscriptionReference;

  /**
   * Type of the line.
   */
  #deprecated "deprecated"
  @visibility(Lifecycle.Read) type: "usage_based";

  /**
   * Price of the usage-based item being sold.
   */
  #deprecated "deprecated"
  price?: V1RateCardUsageBasedPrice;

  /**
   * The feature that the usage is based on.
   */
  #deprecated "deprecated"
  @minLength(1)
  @maxLength(64)
  @pattern("^[a-z0-9]+(?:_[a-z0-9]+)*$")
  feature_key?: string;

  /**
   * The lines detailing the item or service sold.
   */
  @visibility(Lifecycle.Read) children?: V1InvoiceDetailedLine[];

  /**
   * The rate card that is used for this line.
   *
   * The rate card captures the intent of the price and discounts for the usage-based
   * item.
   */
  rate_card?: V1InvoiceUsageBasedRateCard;

  /**
   * The quantity of the item being sold.
   *
   * Any usage discounts applied previously are deducted from this quantity.
   */
  @visibility(Lifecycle.Read) quantity?: V1Numeric;

  /**
   * The quantity of the item that has been metered for the period before any
   * discounts were applied.
   */
  @visibility(Lifecycle.Read) metered_quantity?: V1Numeric;

  /**
   * The quantity of the item used before this line's period.
   *
   * It is non-zero in case of progressive billing, when this shows how much of the
   * usage was already billed.
   *
   * Any usage discounts applied previously are deducted from this quantity.
   */
  @visibility(Lifecycle.Read) pre_line_period_quantity?: V1Numeric;

  /**
   * The metered quantity of the item used in before this line's period without any
   * discounts applied.
   *
   * It is non-zero in case of progressive billing, when this shows how much of the
   * usage was already billed.
   */
  @visibility(Lifecycle.Read) metered_pre_line_period_quantity?: V1Numeric;
}

/**
 * InvoiceLineAmountDiscount represents an amount deducted from the line, and will
 * be applied before taxes.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceLineAmountDiscount")
model V1InvoiceLineAmountDiscount {
  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * ID of the charge or discount.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  id: string;

  /**
   * Reason code.
   */
  @visibility(Lifecycle.Read) reason: V1BillingDiscountReason;

  /**
   * Text description as to why the discount was applied.
   */
  @visibility(Lifecycle.Read) description?: string;

  /**
   * External IDs of the invoice in other apps such as Stripe.
   */
  @visibility(Lifecycle.Read) external_ids?: V1InvoiceLineAppExternalIds;

  /**
   * Fixed discount amount to apply (calculated if percent present).
   */
  @visibility(Lifecycle.Read)
  @summary("Amount in the currency of the invoice")
  amount: V1Numeric;
}

/**
 * InvoiceLineAppExternalIds contains the external IDs of the invoice in other apps
 * such as Stripe.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceLineAppExternalIds")
model V1InvoiceLineAppExternalIds {
  /**
   * The external ID of the invoice in the invoicing app if available.
   */
  @visibility(Lifecycle.Read) invoicing?: string;

  /**
   * The external ID of the invoice in the tax app if available.
   */
  @visibility(Lifecycle.Read) tax?: string;
}

/**
 * InvoiceLineCreditAllocation represents a credit amount allocated to the line
 * before taxes are applied.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceLineCreditAllocation")
model V1InvoiceLineCreditAllocation {
  /**
   * Amount allocated from credits.
   */
  @visibility(Lifecycle.Read)
  @summary("Amount in the currency of the invoice")
  amount: V1Numeric;

  /**
   * Text description as to why the credit was allocated.
   */
  @visibility(Lifecycle.Read) description?: string;
}

/**
 * InvoiceLineDiscounts represents the discounts applied to the invoice line by
 * type.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceLineDiscounts")
model V1InvoiceLineDiscounts {
  /**
   * Amount based discounts applied to the line.
   *
   * Amount based discounts are deduced from the total price of the line.
   */
  amount?: V1InvoiceLineAmountDiscount[];

  /**
   * Usage based discounts applied to the line.
   *
   * Usage based discounts are deduced from the usage of the line before price
   * calculations are applied.
   */
  usage?: V1InvoiceLineUsageDiscount[];
}

/**
 * InvoiceLineManagedBy specifies who manages the line.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceLineManagedBy")
enum V1InvoiceLineManagedBy {
  subscription,
  system,
  manual,
}

/**
 * Line status specifies the status of the line.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceLineStatus")
enum V1InvoiceLineStatus {
  valid,
  detailed,
  split,
}

/**
 * InvoiceLineSubscriptionReference contains the references to the subscription
 * that this line is related to.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceLineSubscriptionReference")
model V1InvoiceLineSubscriptionReference {
  /**
   * The subscription.
   */
  @visibility(Lifecycle.Read) subscription: V1IDResource;

  /**
   * The phase of the subscription.
   */
  @visibility(Lifecycle.Read) phase: V1IDResource;

  /**
   * The item this line is related to.
   */
  @visibility(Lifecycle.Read) item: V1IDResource;

  /**
   * The billing period of the subscription. In case the subscription item's billing
   * period is different from the subscription's billing period, this field will
   * contain the billing period of the subscription itself.
   *
   * For example, in case of:
   *
   * - A monthly billed subscription anchored to 2025-01-01
   * - A subscription item billed daily
   *
   * An example line would have the period of 2025-01-02 to 2025-01-03 as the item is
   * billed daily, but the subscription's billing period will be 2025-01-01 to
   * 2025-01-31.
   */
  @visibility(Lifecycle.Read) billing_period: V1Period;
}

/**
 * InvoiceLineTaxBehavior details how the tax item is applied to the base amount.
 *
 * Inclusive means the tax is included in the base amount. Exclusive means the tax
 * is added to the base amount.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceLineTaxBehavior")
enum V1InvoiceLineTaxBehavior {
  inclusive,
  exclusive,
}

/**
 * TaxConfig stores the configuration for a tax line relative to an invoice line.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceLineTaxItem")
model V1InvoiceLineTaxItem {
  /**
   * Tax provider configuration.
   */
  @visibility(Lifecycle.Read) config?: V1TaxConfig;

  /**
   * Percent defines the percentage set manually or determined from the rate key
   * (calculated if rate present). A nil percent implies that this tax combo is
   * **exempt** from tax.")
   */
  @visibility(Lifecycle.Read) percent?: V1Percentage;

  /**
   * Some countries require an additional surcharge (calculated if rate present).
   */
  @visibility(Lifecycle.Read) surcharge?: V1Numeric;

  /**
   * Is the tax item inclusive or exclusive of the base amount.
   */
  @visibility(Lifecycle.Read) behavior?: V1InvoiceLineTaxBehavior;
}

/**
 * InvoiceLineUsageDiscount represents an usage-based discount applied to the line.
 *
 * The deduction is done before the pricing algorithm is applied.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceLineUsageDiscount")
model V1InvoiceLineUsageDiscount {
  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * ID of the charge or discount.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  id: string;

  /**
   * Reason code.
   */
  @visibility(Lifecycle.Read) reason: V1BillingDiscountReason;

  /**
   * Text description as to why the discount was applied.
   */
  @visibility(Lifecycle.Read) description?: string;

  /**
   * External IDs of the invoice in other apps such as Stripe.
   */
  @visibility(Lifecycle.Read) external_ids?: V1InvoiceLineAppExternalIds;

  /**
   * The usage to apply.
   */
  @visibility(Lifecycle.Read)
  @summary("Usage quantity in the unit of the underlying meter")
  quantity: V1Numeric;

  /**
   * The usage discount already applied to the previous split lines.
   *
   * Only set if progressive billing is enabled and the line is a split line.
   */
  @visibility(Lifecycle.Read)
  @summary("Usage quantity in the unit of the underlying meter")
  pre_line_period_quantity?: V1Numeric;
}

/**
 * InvoiceNumber is a unique identifier for the invoice, generated by the invoicing
 * app.
 *
 * The uniqueness depends on a lot of factors:
 *
 * - app setting (unique per app or unique per customer)
 * - multiple app scenarios (multiple apps generating invoices with the same
 *   prefix)
 */
@minLength(1)
@maxLength(256)
@friendlyName("BillingNotificationEventV1InvoiceNumber")
scalar V1InvoiceNumber extends string;

/**
 * Payment contains details as to how the invoice should be paid.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoicePaymentTerms")
model V1InvoicePaymentTerms {
  /**
   * The terms of payment for the invoice.
   */
  terms?: V1PaymentTerms;
}

/**
 * Reference to an invoice.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceReference")
model V1InvoiceReference {
  /**
   * The ID of the invoice.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  id: string;

  /**
   * The number of the invoice.
   */
  @visibility(Lifecycle.Read) number?: V1InvoiceNumber;
}

/**
 * InvoiceStatus describes the status of an invoice.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceStatus")
enum V1InvoiceStatus {
  gathering,
  draft,
  issuing,
  issued,
  payment_processing,
  overdue,
  paid,
  uncollectible,
  voided,
}

/**
 * InvoiceStatusDetails represents the details of the invoice status.
 *
 * API users are encouraged to rely on the immutable/failed/avaliableActions fields
 * to determine the next steps of the invoice instead of the extendedStatus field.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceStatusDetails")
model V1InvoiceStatusDetails {
  /**
   * Is the invoice editable?
   */
  @visibility(Lifecycle.Read) immutable: boolean;

  /**
   * Is the invoice in a failed state?
   */
  @visibility(Lifecycle.Read) failed: boolean;

  /**
   * Extended status information for the invoice.
   */
  @visibility(Lifecycle.Read) extended_status: string;

  /**
   * The actions that can be performed on the invoice.
   */
  available_actions: V1InvoiceAvailableActions;
}

/**
 * Totals contains the summaries of all calculations for the invoice.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "Copied v1 field layout."
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceTotals")
model V1InvoiceTotals {
  /**
   * The total value of the line before taxes, discounts and commitments.
   */
  @visibility(Lifecycle.Read) amount: V1Numeric;

  /**
   * The amount of value of the line that are due to additional charges.
   */
  @visibility(Lifecycle.Read) charges_total: V1Numeric;

  /**
   * The amount of value of the line that are due to discounts.
   */
  @visibility(Lifecycle.Read) discounts_total: V1Numeric;

  /**
   * The amount of value of the line that are due to credits.
   */
  @visibility(Lifecycle.Read) credits_total: V1Numeric;

  /**
   * The total amount of taxes that are included in the line.
   */
  @visibility(Lifecycle.Read) taxes_inclusive_total: V1Numeric;

  /**
   * The total amount of taxes that are added on top of amount from the line.
   */
  @visibility(Lifecycle.Read) taxes_exclusive_total: V1Numeric;

  /**
   * The total amount of taxes for this line.
   */
  @visibility(Lifecycle.Read) taxes_total: V1Numeric;

  /**
   * The total amount value of the line after taxes, discounts and commitments.
   */
  @visibility(Lifecycle.Read) total: V1Numeric;
}

/**
 * InvoiceType represents the type of invoice.
 *
 * The type of invoice determines the purpose of the invoice and how it should be
 * handled.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceType")
enum V1InvoiceType {
  standard,
  credit_note,
}

/**
 * InvoiceUsageBasedRateCard represents the rate card (intent) for an usage-based
 * line.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
#suppress "@openmeter/api-spec-aip/no-nullable" "Legacy v1 wire shape: the field is required and nullable."
@friendlyName("BillingNotificationEventV1InvoiceUsageBasedRateCard")
model V1InvoiceUsageBasedRateCard {
  /**
   * The feature the customer is entitled to use.
   */
  @minLength(1)
  @maxLength(64)
  @pattern("^[a-z0-9]+(?:_[a-z0-9]+)*$")
  @summary("Feature key")
  feature_key?: string;

  /**
   * The tax config of the rate card. When undefined, the tax config of the feature
   * or the default tax config of the plan is used.
   */
  @summary("Tax config") tax_config?: V1TaxConfig;

  /**
   * The price of the rate card. When null, the feature or service is free.
   */
  price: V1RateCardUsageBasedPrice | null;

  /**
   * The discounts that are applied to the line.
   */
  #deprecated "deprecated"
  discounts?: V1BillingDiscounts;
}

/**
 * InvoiceWorkflowSettings represents the workflow settings used by the invoice.
 *
 * This is a clone of the billing profile's workflow settings at the time of
 * invoice creation with customer overrides considered.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1InvoiceWorkflowSettings")
model V1InvoiceWorkflowSettings {
  /**
   * The apps that will be used to orchestrate the invoice's workflow.
   *
   * The v1 schema allows app references here too, but the v1 Event payloads always
   * carry the full apps.
   */
  @visibility(Lifecycle.Read) apps?: V1BillingProfileApps;

  /**
   * sourceBillingProfileID is the billing profile on which the workflow was based
   * on.
   *
   * The profile is snapshotted on invoice creation, after which it can be altered
   * independently of the profile itself.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  source_billing_profile_id: string;

  /**
   * The workflow details used by this invoice.
   */
  workflow: V1BillingWorkflow;
}

/**
 * A marketplace listing. Represent an available app in the app marketplace that
 * can be installed to the organization.
 *
 * Marketplace apps only exist in config so they don't extend the Resource model.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1MarketplaceListing")
model V1MarketplaceListing {
  /**
   * The app's type
   */
  type: V1AppType;

  /**
   * The app's name.
   */
  name: string;

  /**
   * The app's description.
   */
  description: string;

  /**
   * The app's capabilities.
   */
  capabilities: V1AppCapability[];

  /**
   * Install methods.
   *
   * List of methods to install the app.
   */
  install_methods: V1InstallMethod[];
}

/**
 * Set of key-value pairs. Metadata can be used to store additional information
 * about a resource.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@extension("x-go-type", "map[string]string")
@friendlyName("BillingNotificationEventV1Metadata")
model V1Metadata {
  ...Record<string>;
}

/**
 * Data of the payload for notification event with `entitlements.balance.threshold`
 * type.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1BalanceThresholdPayloadData")
model V1NotificationEventBalanceThresholdPayloadData {
  /**
   * Entitlement.
   */
  #suppress "deprecated" "The v1 payload still carries this deprecated v1 model."
  @visibility(Lifecycle.Read)
  @summary("Entitlement")
  entitlement: V1EntitlementMetered;

  /**
   * Feature.
   */
  @visibility(Lifecycle.Read) @summary("Feature") feature: V1Feature;

  /**
   * Subject.
   */
  #suppress "deprecated" "The v1 payload still carries this deprecated v1 model."
  @visibility(Lifecycle.Read) @summary("Subject") subject: V1Subject;

  /**
   * Entitlement Value.
   */
  @visibility(Lifecycle.Read)
  @summary("Entitlement Value")
  value: V1EntitlementValue;

  /**
   * Customer.
   */
  @visibility(Lifecycle.Read) @summary("Customer") customer?: V1Customer;

  /**
   * Threshold.
   */
  @visibility(Lifecycle.Read)
  @summary("Threshold")
  threshold: V1NotificationRuleBalanceThresholdValue;
}

/**
 * Base data for any payload with entitlement entitlement value.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1EntitlementValuePayloadBase")
model V1NotificationEventEntitlementValuePayloadBase {
  /**
   * Entitlement.
   */
  #suppress "deprecated" "The v1 payload still carries this deprecated v1 model."
  @visibility(Lifecycle.Read)
  @summary("Entitlement")
  entitlement: V1EntitlementMetered;

  /**
   * Feature.
   */
  @visibility(Lifecycle.Read) @summary("Feature") feature: V1Feature;

  /**
   * Subject.
   */
  #suppress "deprecated" "The v1 payload still carries this deprecated v1 model."
  @visibility(Lifecycle.Read) @summary("Subject") subject: V1Subject;

  /**
   * Entitlement Value.
   */
  @visibility(Lifecycle.Read)
  @summary("Entitlement Value")
  value: V1EntitlementValue;

  /**
   * Customer.
   */
  @visibility(Lifecycle.Read) @summary("Customer") customer?: V1Customer;
}

/**
 * Threshold value with multiple supported types.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1RuleBalanceThresholdValue")
model V1NotificationRuleBalanceThresholdValue {
  /**
   * Value of the threshold.
   */
  @summary("Threshold Value") value: float64;

  /**
   * Type of the threshold.
   */
  type: V1NotificationRuleBalanceThresholdValueType;
}

/**
 * Type of the rule in the balance threshold specification:
 *
 * - `balance_value`: threshold defined by the remaining balance value based on
 *   usage and the total of grants in the current usage period
 * - `usage_percentage`: threshold defined by the usage percentage compared to the
 *   total of grants in the current usage period
 * - `usage_value`: threshold defined by the usage value in the current usage
 *   period
 * - `NUMBER` (**deprecated**): see `usage_value`
 * - `PERCENT` (**deprecated**): see `usage_percentage`
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@extension(
  "x-enum-varnames",
  #["Percent", "Number", "BalanceValue", "UsagePercentage", "UsageValue"]
)
@summary("Notification balance threshold type")
@friendlyName("BillingNotificationEventV1RuleBalanceThresholdValueType")
enum V1NotificationRuleBalanceThresholdValueType {
  PERCENT,
  NUMBER,
  balance_value,
  usage_percentage,
  usage_value,
}

/**
 * Numeric represents an arbitrary precision number.
 */
@pattern("^\\-?[0-9]+(\\.[0-9]+)?$")
@friendlyName("BillingNotificationEventV1Numeric")
scalar V1Numeric extends string;

/**
 * Package price with spend commitments.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1PackagePriceWithCommitments")
model V1PackagePriceWithCommitments {
  /**
   * The type of the price.
   */
  type: "package";

  /**
   * The price of one package.
   */
  @summary("Amount") amount: V1Numeric;

  /**
   * The quantity per package.
   */
  @summary("Quantity per package") quantity_per_package: V1Numeric;

  /**
   * The customer is committed to spend at least the amount.
   */
  @summary("Minimum amount") minimum_amount?: V1Numeric;

  /**
   * The customer is limited to spend at most the amount.
   */
  @summary("Maximum amount") maximum_amount?: V1Numeric;
}

/**
 * PaymentDueDate contains an amount that should be paid by the given date.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1PaymentDueDate")
model V1PaymentDueDate {
  /**
   * When the payment is due.
   */
  @visibility(Lifecycle.Read) due_at: utcDateTime;

  /**
   * Other details to take into account for the due date.
   */
  @visibility(Lifecycle.Read) notes?: string;

  /**
   * How much needs to be paid by the date.
   */
  @visibility(Lifecycle.Read) amount: V1Numeric;

  /**
   * Percentage of the total that should be paid by the date.
   */
  @visibility(Lifecycle.Read) percent?: V1Percentage;

  /**
   * If different from the parent document's base currency.
   */
  @visibility(Lifecycle.Read) currency?: V1CurrencyCode;
}

/**
 * PaymentTermDueDate defines the terms for payment on a specific date.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1PaymentTermDueDate")
model V1PaymentTermDueDate {
  /**
   * Type of terms to be applied.
   */
  type: "due_date";

  /**
   * Text detail of the chosen payment terms.
   */
  @visibility(Lifecycle.Read) detail?: string;

  /**
   * Description of the conditions for payment.
   */
  @visibility(Lifecycle.Read) notes?: string;

  /**
   * When the payment is due.
   */
  @visibility(Lifecycle.Read) @minItems(1) due_at: V1PaymentDueDate[];
}

/**
 * PaymentTermInstant defines the terms for payment on receipt of invoice.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1PaymentTermInstant")
model V1PaymentTermInstant {
  /**
   * Type of terms to be applied.
   */
  type: "instant";

  /**
   * Text detail of the chosen payment terms.
   */
  @visibility(Lifecycle.Read) detail?: string;

  /**
   * Description of the conditions for payment.
   */
  @visibility(Lifecycle.Read) notes?: string;
}

/**
 * PaymentTerms defines the terms for payment.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1PaymentTerms")
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union V1PaymentTerms {
  instant: V1PaymentTermInstant,
  due_date: V1PaymentTermDueDate,
}

/**
 * Numeric representation of a percentage
 *
 * 50% is represented as 50
 */
@extension("x-go-package", "github.com/openmeterio/openmeter/pkg/models")
@extension("x-go-type", "models.Percentage")
@friendlyName("BillingNotificationEventV1Percentage")
scalar V1Percentage extends float64;

/**
 * A period with a start and end time.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1Period")
model V1Period {
  /**
   * Period start time.
   */
  from: utcDateTime;

  /**
   * Period end time.
   */
  to: utcDateTime;
}

/**
 * References an exact plan.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1PlanReference")
model V1PlanReference {
  /**
   * The plan ID.
   */
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$") id: string;

  /**
   * The plan key.
   */
  @minLength(1)
  @maxLength(64)
  @pattern("^[a-z0-9]+(?:_[a-z0-9]+)*$")
  key: string;

  /**
   * The plan version.
   */
  version: integer;
}

/**
 * The payment term of a flat price. One of: in_advance or in_arrears.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1PricePaymentTerm")
enum V1PricePaymentTerm {
  in_advance,
  in_arrears,
}

/**
 * A price tier. At least one price component is required in each tier.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
#suppress "@openmeter/api-spec-aip/no-nullable" "Legacy v1 wire shape: the field is required and nullable."
@friendlyName("BillingNotificationEventV1PriceTier")
model V1PriceTier {
  /**
   * Up to and including to this quantity will be contained in the tier. If null, the
   * tier is open-ended.
   */
  @summary("Up to quantity") up_to_amount?: V1Numeric;

  /**
   * The flat price component of the tier.
   */
  @summary("Flat price component") flat_price: V1FlatPrice | null;

  /**
   * The unit price component of the tier.
   */
  @summary("Unit price component") unit_price: V1UnitPrice | null;
}

/**
 * Configuration for pro-rating behavior.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1ProRatingConfig")
model V1ProRatingConfig {
  /**
   * Whether pro-rating is enabled for this plan.
   */
  @summary("Enable pro-rating") enabled: boolean;

  /**
   * How to handle pro-rating for billing period changes.
   */
  @summary("Pro-rating mode")
  mode: V1ProRatingMode;
}

/**
 * Pro-rating mode options for handling billing period changes.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1ProRatingMode")
enum V1ProRatingMode {
  prorate_prices,
}

/**
 * The price of the usage based rate card.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
@oneOf
@friendlyName("BillingNotificationEventV1RateCardUsageBasedPrice")
union V1RateCardUsageBasedPrice {
  /**
   * flat.
   */
  flat: V1FlatPriceWithPaymentTerm,

  /**
   * unit.
   */
  unit: V1UnitPriceWithCommitments,

  /**
   * tiered.
   */
  tiered: V1TieredPriceWithCommitments,

  /**
   * dynamic.
   */
  dynamic: V1DynamicPriceWithCommitments,

  /**
   * package.
   */
  package: V1PackagePriceWithCommitments,
}

/**
 * Recurring period with an interval and an anchor.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
#deprecated "deprecated"
@friendlyName("BillingNotificationEventV1RecurringPeriod")
model V1RecurringPeriod {
  /**
   * The unit of time for the interval. Heuristically maps ISO duraitons to enum
   * values or returns the ISO duration.
   */
  @summary("Interval") interval: V1RecurringPeriodInterval;

  /**
   * A date-time anchor to base the recurring period on.
   */
  @summary("Anchor time") anchor: utcDateTime;

  /**
   * The unit of time for the interval in ISO8601 format.
   */
  interval_iso: duration;
}

/**
 * Period duration for the recurrence
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1RecurringPeriodInterval")
union V1RecurringPeriodInterval {
  string,
  V1RecurringPeriodIntervalEnum,
}

/**
 * The unit of time for the interval. One of: `day`, `week`, `month`, or `year`.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1RecurringPeriodIntervalEnum")
enum V1RecurringPeriodIntervalEnum {
  DAY,
  WEEK,
  MONTH,
  YEAR,
}

/**
 * Recurring period with an interval and an anchor.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1RecurringPeriodV2")
model V1RecurringPeriodV2 {
  /**
   * The unit of time for the interval. Heuristically maps ISO duraitons to enum
   * values or returns the ISO duration.
   */
  @summary("Interval") interval: V1RecurringPeriodInterval;

  /**
   * A date-time anchor to base the recurring period on.
   */
  @summary("Anchor time") anchor: utcDateTime;
}

/**
 * Sandbox app can be used for testing OpenMeter features.
 *
 * The app is not creating anything in external systems, thus it is safe to use for
 * verifying OpenMeter features.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1SandboxApp")
model V1SandboxApp {
  /**
   * A unique identifier for the resource.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  @summary("ID")
  id: string;

  /**
   * Human-readable name for the resource. Between 1 and 256 characters.
   */
  @minLength(1)
  @maxLength(256)
  @summary("Display name")
  name: string;

  /**
   * Optional description of the resource. Maximum 1024 characters.
   */
  @maxLength(1024) @summary("Description") description?: string;

  /**
   * Additional metadata for the resource.
   */
  @summary("Metadata") metadata?: V1Metadata | null;

  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * The marketplace listing that this installed app is based on.
   */
  @visibility(Lifecycle.Read) listing: V1MarketplaceListing;

  /**
   * Status of the app connection.
   */
  @visibility(Lifecycle.Read) status: V1AppStatus;

  /**
   * The app's type is Sandbox.
   */
  type: "sandbox";
}

/**
 * A installed Stripe app object.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1StripeApp")
model V1StripeApp {
  /**
   * A unique identifier for the resource.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  @summary("ID")
  id: string;

  /**
   * Human-readable name for the resource. Between 1 and 256 characters.
   */
  @minLength(1)
  @maxLength(256)
  @summary("Display name")
  name: string;

  /**
   * Optional description of the resource. Maximum 1024 characters.
   */
  @maxLength(1024) @summary("Description") description?: string;

  /**
   * Additional metadata for the resource.
   */
  @summary("Metadata") metadata?: V1Metadata | null;

  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * The marketplace listing that this installed app is based on.
   */
  @visibility(Lifecycle.Read) listing: V1MarketplaceListing;

  /**
   * Status of the app connection.
   */
  @visibility(Lifecycle.Read) status: V1AppStatus;

  /**
   * The app's type is Stripe.
   */
  type: "stripe";

  /**
   * The Stripe account ID.
   */
  @visibility(Lifecycle.Read) stripe_account_id: string;

  /**
   * Livemode, true if the app is in production mode.
   */
  @visibility(Lifecycle.Read) livemode: boolean;

  /**
   * The masked API key. Only shows the first 8 and last 3 characters.
   */
  @visibility(Lifecycle.Read) masked_api_key: string;
}

/**
 * The tax config for Stripe.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1StripeTaxConfig")
model V1StripeTaxConfig {
  /**
   * Product tax code.
   *
   * See: https://docs.stripe.com/tax/tax-codes
   */
  @pattern("^txcd_\\d{8}$") @summary("Tax code") code: string;
}

/**
 * A subject is a unique identifier for a usage attribution by its key. Subjects
 * only exist in the concept of metering. Subjects are optional to create and work
 * as an enrichment for the subject key like displayName, metadata, etc. Subjects
 * are useful when you are reporting usage events with your own database ID but
 * want to enrich the subject with a human-readable name or metadata. For most use
 * cases, a subject is equivalent to a customer.
 *
 * ⚠️ **Deprecated**: Subjects as managable entities are being depracated, use
 * customers with subject key usage attribution instead.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "Copied v1 field layout."
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
#deprecated "deprecated"
@friendlyName("BillingNotificationEventV1Subject")
model V1Subject {
  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * A unique identifier for the subject.
   */
  @visibility(Lifecycle.Read)
  id: string;

  /**
   * A unique, human-readable identifier for the subject. This is typically a
   * database ID or a customer key.
   */
  key: string;

  /**
   * A human-readable display name for the subject.
   */
  display_name?: string | null;

  /**
   * Metadata for the subject.
   */
  metadata?: Record<unknown> | null;

  /**
   * The start of the current period for the subject.
   */
  #deprecated "deprecated"
  current_period_start?: utcDateTime;

  /**
   * The end of the current period for the subject.
   */
  #deprecated "deprecated"
  current_period_end?: utcDateTime;

  /**
   * The Stripe customer ID for the subject.
   */
  #deprecated "deprecated"
  stripe_customer_id?: string | null;
}

/**
 * Subscription is an exact subscription instance.
 */
#suppress "@openmeter/api-spec-aip/repeated-prefix-grouping" "Copied v1 field layout."
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1Subscription")
model V1Subscription {
  /**
   * A unique identifier for the resource.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  @summary("ID")
  id: string;

  /**
   * Human-readable name for the resource. Between 1 and 256 characters.
   */
  @minLength(1)
  @maxLength(256)
  @summary("Display name")
  name: string;

  /**
   * Optional description of the resource. Maximum 1024 characters.
   */
  @maxLength(1024) @summary("Description") description?: string;

  /**
   * Additional metadata for the resource.
   */
  @summary("Metadata") metadata?: V1Metadata | null;

  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * The cadence start of the resource.
   */
  active_from: utcDateTime;

  /**
   * The cadence end of the resource.
   */
  active_to?: utcDateTime;

  /**
   * Set of key-value pairs managed by the system. Cannot be modified by user.
   */
  @visibility(Lifecycle.Read)
  @summary("Annotations")
  annotations?: V1Annotations;

  /**
   * Alignment configuration for the plan.
   */
  #suppress "deprecated" "The v1 payload still carries this deprecated v1 model."
  alignment?: V1Alignment;

  /**
   * The status of the subscription.
   */
  @visibility(Lifecycle.Read) status: V1SubscriptionStatus;

  /**
   * The customer ID of the subscription.
   */
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$") customer_id: string;

  /**
   * The plan of the subscription.
   */
  plan?: V1PlanReference;

  /**
   * The fiat currency in which the subscription is invoiced.
   */
  @summary("Currency") currency: V1CurrencyCode;

  /**
   * Controls whether custom-currency cost bases are resolved dynamically or pinned
   * when their currency pair is introduced to the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Cost basis mode")
  cost_basis_mode: V1SubscriptionCostBasisMode;

  /**
   * Cost bases pinned to custom-currency pairs for this subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Pinned cost bases")
  cost_basis_pins: V1SubscriptionCostBasisPin[];

  /**
   * The billing cadence for the subscriptions. Defines how often customers are
   * billed using ISO8601 duration format. Examples: "P1M" (monthly), "P3M"
   * (quarterly), "P1Y" (annually).
   */
  @visibility(Lifecycle.Read)
  @summary("Billing cadence")
  billing_cadence: duration;

  /**
   * The pro-rating configuration for the subscriptions.
   */
  @visibility(Lifecycle.Read)
  @summary("Pro-rating configuration")
  pro_rating_config?: V1ProRatingConfig;

  /**
   * The normalizedbilling anchor of the subscription.
   */
  @visibility(Lifecycle.Read)
  @summary("Billing anchor")
  billing_anchor: utcDateTime;

  /**
   * The settlement mode of the subscription.
   *
   * - credit_then_invoice: credits from the previous billing period are applied
   *   first, then the remaining balance is invoiced.
   * - credit_only: only credits from the previous billing period are generated and
   *   applied. No invoices are generated for the subscription. This is the default
   *   and most common settlement mode.
   */
  @visibility(Lifecycle.Read)
  @summary("Settlement mode")
  settlement_mode: V1BillingSettlementMode;
}

/**
 * Controls how custom-currency cost bases are selected for the subscription.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1SubscriptionCostBasisMode")
enum V1SubscriptionCostBasisMode {
  dynamic,
  pinned,
}

/**
 * A cost basis pinned to a custom-currency pair for the subscription.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1SubscriptionCostBasisPin")
model V1SubscriptionCostBasisPin {
  /**
   * The managed custom currency ID.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  custom_currency_id: string;

  /**
   * The fiat currency in which the subscription is invoiced.
   */
  @visibility(Lifecycle.Read) invoice_currency: V1CurrencyCode;

  /**
   * The pinned cost basis resource ID.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  cost_basis_id: string;
}

/**
 * Subscription status.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1SubscriptionStatus")
enum V1SubscriptionStatus {
  active,
  inactive,
  canceled,
  scheduled,
}

/**
 * Tax behavior.
 *
 * This enum is used to specify whether tax is included in the price or excluded
 * from the price.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1TaxBehavior")
enum V1TaxBehavior {
  inclusive,
  exclusive,
}

/**
 * Set of provider specific tax configs.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1TaxConfig")
model V1TaxConfig {
  /**
   * Tax behavior.
   *
   * If not specified the billing profile is used to determine the tax behavior. If
   * not specified in the billing profile, the provider's default behavior is used.
   */
  @summary("Tax behavior") behavior?: V1TaxBehavior;

  /**
   * Stripe tax config.
   */
  #deprecated "deprecated"
  @summary("Stripe tax config") stripe?: V1StripeTaxConfig;

  /**
   * Custom invoicing tax config.
   */
  #deprecated "deprecated"
  @summary("Custom invoicing tax config")
  custom_invoicing?: V1CustomInvoicingTaxConfig;

  /**
   * Tax code reference.
   *
   * When both `taxCodeId` and `stripe.code` are provided, `taxCodeId` takes
   * precedence: the referenced tax code entity is used and `stripe.code` is ignored.
   */
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  @summary("Tax code ID")
  tax_code_id?: string;
}

/**
 * The mode of the tiered price.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1TieredPriceMode")
enum V1TieredPriceMode {
  volume,
  graduated,
}

/**
 * Tiered price with spend commitments.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1TieredPriceWithCommitments")
model V1TieredPriceWithCommitments {
  /**
   * The type of the price.
   *
   * One of: flat, unit, or tiered.
   */
  type: "tiered";

  /**
   * Defines if the tiering mode is volume-based or graduated:
   *
   * - In `volume`-based tiering, the maximum quantity within a period determines the
   *   per unit price.
   * - In `graduated` tiering, pricing can change as the quantity grows.
   */
  @summary("Mode") mode: V1TieredPriceMode;

  /**
   * The tiers of the tiered price. At least one price component is required in each
   * tier.
   */
  @minItems(1) @summary("Tiers") tiers: V1PriceTier[];

  /**
   * The customer is committed to spend at least the amount.
   */
  @summary("Minimum amount") minimum_amount?: V1Numeric;

  /**
   * The customer is limited to spend at most the amount.
   */
  @summary("Maximum amount") maximum_amount?: V1Numeric;
}

/**
 * Unit price.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1UnitPrice")
model V1UnitPrice {
  /**
   * The type of the price.
   */
  type: "unit";

  /**
   * The amount of the unit price.
   */
  amount: V1Numeric;
}

/**
 * Unit price with spend commitments.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1UnitPriceWithCommitments")
model V1UnitPriceWithCommitments {
  /**
   * The type of the price.
   */
  type: "unit";

  /**
   * The amount of the unit price.
   */
  amount: V1Numeric;

  /**
   * The customer is committed to spend at least the amount.
   */
  @summary("Minimum amount") minimum_amount?: V1Numeric;

  /**
   * The customer is limited to spend at most the amount.
   */
  @summary("Maximum amount") maximum_amount?: V1Numeric;
}

/**
 * ValidationIssue captures any validation issues related to the invoice.
 *
 * Issues with severity "critical" will prevent the invoice from being issued.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1ValidationIssue")
model V1ValidationIssue {
  /**
   * Timestamp of when the resource was created.
   */
  @visibility(Lifecycle.Read) @summary("Creation Time") created_at: utcDateTime;

  /**
   * Timestamp of when the resource was last updated.
   */
  @visibility(Lifecycle.Read)
  @summary("Last Update Time")
  updated_at: utcDateTime;

  /**
   * Timestamp of when the resource was permanently deleted.
   */
  @visibility(Lifecycle.Read)
  @summary("Deletion Time")
  deleted_at?: utcDateTime;

  /**
   * ID of the charge or discount.
   */
  @visibility(Lifecycle.Read)
  @pattern("^[0-7][0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{25}$")
  id: string;

  /**
   * The severity of the issue.
   */
  @visibility(Lifecycle.Read) severity: V1ValidationIssueSeverity;

  /**
   * The field that the issue is related to, if available in JSON path format.
   */
  @visibility(Lifecycle.Read) field?: string;

  /**
   * Machine indentifiable code for the issue, if available.
   */
  @visibility(Lifecycle.Read) code?: string;

  /**
   * Component reporting the issue.
   */
  @visibility(Lifecycle.Read) component: string;

  /**
   * A human-readable description of the issue.
   */
  @visibility(Lifecycle.Read) message: string;

  /**
   * Additional context for the issue.
   */
  @visibility(Lifecycle.Read) metadata?: V1Metadata;
}

/**
 * ValidationIssueSeverity describes the severity of a validation issue.
 *
 * Issues with severity "critical" will prevent the invoice from being issued.
 */
#suppress "@openmeter/api-spec-aip/casing" "Copied v1 type and enum member names."
@friendlyName("BillingNotificationEventV1ValidationIssueSeverity")
enum V1ValidationIssueSeverity {
  critical,
  warning,
}
