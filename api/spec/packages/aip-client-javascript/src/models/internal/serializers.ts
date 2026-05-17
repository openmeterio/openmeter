import type {
  Addon,
  Address,
  Address_2,
  App,
  AppCatalogItem,
  AppCustomerData,
  AppCustomerDataExternalInvoicing,
  AppCustomerDataStripe,
  AppExternalInvoicing,
  AppReference,
  AppSandbox,
  AppStripe,
  BillingParty,
  BillingPartyAddresses,
  BillingPartyTaxIdentity,
  BillingProfile,
  BillingProfileAppReferences,
  BillingProfileReference,
  BillingTotals,
  BillingWorkflow,
  BillingWorkflowCollectionAlignment,
  BillingWorkflowCollectionAlignmentAnchored,
  BillingWorkflowCollectionAlignmentSubscription,
  BillingWorkflowCollectionSettings,
  BillingWorkflowInvoicingSettings,
  BillingWorkflowPaymentChargeAutomaticallySettings,
  BillingWorkflowPaymentSendInvoiceSettings,
  BillingWorkflowPaymentSettings,
  BillingWorkflowTaxSettings,
  BooleanFieldFilter,
  Charge,
  ChargesExpand,
  ChargeTotals,
  CheckoutSessionCustomTextParams,
  ClosedPeriod,
  CostBasis,
  CreateCheckoutSessionTaxIdCollection,
  CreateRequest_11 as CreateRequest,
  CreateRequest_2 as CreateRequest_10,
  CreateRequest as CreateRequest_11,
  CreateRequest_10 as CreateRequest_2,
  CreateRequest_9 as CreateRequest_3,
  CreateRequest_8 as CreateRequest_4,
  CreateRequest_7 as CreateRequest_5,
  CreateRequest_6,
  CreateRequest_5 as CreateRequest_7,
  CreateRequest_4 as CreateRequest_8,
  CreateRequest_3 as CreateRequest_9,
  CreateRequestNested,
  CreateStripeCheckoutSessionConsentCollection,
  CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement,
  CreateStripeCheckoutSessionCustomerUpdate,
  CreateStripeCheckoutSessionRequestOptions,
  CreateStripeCheckoutSessionResult,
  CreateStripeCustomerPortalSessionOptions,
  CreateStripeCustomerPortalSessionResult,
  CreditAdjustment,
  CreditBalance,
  CreditBalances,
  CreditGrant,
  CreditGrantTaxConfig,
  CreditTransaction,
  Currency,
  CurrencyAmount,
  CurrencyCode_2 as CurrencyCode,
  CurrencyCustom,
  CurrencyFiat,
  CursorMeta,
  CursorMetaPage,
  CursorPaginationQueryPage,
  Customer,
  CustomerBillingData,
  CustomerBillingStripeCreateCheckoutSessionRequest,
  CustomerBillingStripeCreateCustomerPortalSessionRequest,
  CustomerReference,
  CustomerUsageAttribution,
  DateTimeFieldFilter,
  Discounts,
  EntitlementAccessResult,
  Feature,
  FeatureCostQueryResult,
  FeatureCostQueryRow,
  FeatureLlmUnitCost,
  FeatureLlmUnitCostPricing,
  FeatureManualUnitCost,
  FeatureMeterReference,
  FeatureUnitCost,
  FeatureUpdateRequest,
  FieldFilters,
  FlatFeeCharge,
  FlatFeeDiscounts,
  GetCreditBalanceParamsFilter,
  GovernanceFeatureAccess,
  GovernanceFeatureAccessReason,
  GovernanceQueryError,
  GovernanceQueryRequest,
  GovernanceQueryRequestCustomers,
  GovernanceQueryRequestFeatures,
  GovernanceQueryResponse,
  GovernanceQueryResult,
  IngestedEvent,
  IngestedEventValidationError,
  Labels,
  ListAddonsParamsFilter,
  ListCostBasesParamsFilter,
  ListCreditGrantsParamsFilter,
  ListCreditTransactionsParamsFilter,
  ListCurrenciesParamsFilter,
  ListCustomerChargesParamsFilter,
  ListCustomerEntitlementAccessResponseData,
  ListCustomersParamsFilter,
  ListEventsParamsFilter,
  ListFeaturesParamsFilter,
  ListMetersParamsFilter,
  ListPlansParamsFilter,
  ListPricesParamsFilter,
  ListSubscriptionsParamsFilter,
  Meter,
  MeterAggregation,
  MeteringEvent,
  MeterQueryFilters,
  MeterQueryGranularity,
  MeterQueryRequest,
  MeterQueryResult,
  MeterQueryRow,
  Model,
  ModelPricing,
  NumericFieldFilter,
  OrganizationDefaultTaxCodes,
  OverrideCreate,
  PageMeta,
  PagePaginatedMeta,
  Plan,
  PlanAddon,
  PlanPhase,
  Price,
  Price_2,
  PriceFlat,
  PriceFree,
  PriceGraduated,
  PricePaymentTerm,
  PriceTier,
  PriceUnit,
  PriceVolume,
  ProductCatalogValidationError,
  ProrationConfiguration,
  Provider,
  QueryFilterString,
  QueryFilterStringMapItem,
  RateCard,
  RateCardTaxConfig,
  RecurringPeriod,
  ResourceReference,
  ResourceReference_2,
  ResourceReference_3,
  ResourceReference_4,
  SortQuery,
  SpendCommitments,
  StringFieldFilter,
  StringFieldFilterExact,
  Subscription,
  SubscriptionAddon,
  SubscriptionCancel,
  SubscriptionChange,
  SubscriptionChangeResponse,
  SubscriptionCreate,
  SubscriptionEditTiming,
  SubscriptionReference,
  TaxCode,
  TaxCodeAppMapping,
  TaxCodeReference,
  TaxConfig,
  TaxConfigExternalInvoicing,
  TaxConfigStripe,
  UlidFieldFilter,
  UpdateCreditGrantExternalSettlementRequest,
  UpdateRequest_2 as UpdateRequest,
  UpdateRequest as UpdateRequest_2,
  UpsertRequest_8 as UpsertRequest,
  UpsertRequest_7 as UpsertRequest_2,
  UpsertRequest_6 as UpsertRequest_3,
  UpsertRequest_5 as UpsertRequest_4,
  UpsertRequest_4 as UpsertRequest_5,
  UpsertRequest_2 as UpsertRequest_6,
  UpsertRequest_3 as UpsertRequest_7,
  UpsertRequest as UpsertRequest_8,
  UsageBasedCharge,
} from "../models.js";

export function decodeBase64(value: string): Uint8Array | undefined {
  if(!value) {
    return value as any;
  }
  // Normalize Base64URL to Base64
  const base64 = value.replace(/-/g, '+').replace(/_/g, '/')
    .padEnd(value.length + (4 - (value.length % 4)) % 4, '=');

  return new Uint8Array(Buffer.from(base64, 'base64'));
}export function encodeUint8Array(
  value: Uint8Array | undefined | null,
  encoding: BufferEncoding,
): string | undefined {
  if (!value) {
    return value as any;
  }
  return Buffer.from(value).toString(encoding);
}export function dateDeserializer(date?: string | null): Date {
  if (!date) {
    return date as any;
  }

  return new Date(date);
}export function dateRfc7231Deserializer(date?: string | null): Date {
  if (!date) {
    return date as any;
  }

  return new Date(date);
}export function dateRfc3339Serializer(date?: Date | null): string {
  if (!date) {
    return date as any
  }

  return date.toISOString();
}export function dateRfc7231Serializer(date?: Date | null): string {
  if (!date) {
    return date as any;
  }

  return date.toUTCString();
}export function dateUnixTimestampSerializer(date?: Date | null): number {
  if (!date) {
    return date as any;
  }

  return Math.floor(date.getTime() / 1000);
}export function dateUnixTimestampDeserializer(date?: number | null): Date {
  if (!date) {
    return date as any;
  }

  return new Date(date * 1000);
}export function queryPayloadToTransport(payload: GovernanceQueryRequest) {
  return jsonGovernanceQueryRequestToTransportTransform(payload)!;
}export function updatePayloadToTransport(payload: UpdateRequest) {
  return jsonUpdateRequestToTransportTransform_2(payload)!;
}export function createPlanAddonPayloadToTransport(payload: CreateRequest) {
  return jsonCreateRequestToTransportTransform_11(payload)!;
}export function updatePlanAddonPayloadToTransport(payload: UpsertRequest) {
  return jsonUpsertRequestToTransportTransform_8(payload)!;
}export function createAddonPayloadToTransport(payload: CreateRequest_2) {
  return jsonCreateRequestToTransportTransform_10(payload)!;
}export function updateAddonPayloadToTransport(payload: UpsertRequest_2) {
  return jsonUpsertRequestToTransportTransform_7(payload)!;
}export function createPlanPayloadToTransport(payload: CreateRequest_3) {
  return jsonCreateRequestToTransportTransform_9(payload)!;
}export function updatePlanPayloadToTransport(payload: UpsertRequest_3) {
  return jsonUpsertRequestToTransportTransform_6(payload)!;
}export function createOverridePayloadToTransport(payload: OverrideCreate) {
  return jsonOverrideCreateToTransportTransform(payload)!;
}export function queryCostPayloadToTransport(payload: MeterQueryRequest) {
  return jsonMeterQueryRequestToTransportTransform(payload)!;
}export function createPayloadToTransport(payload: CreateRequest_4) {
  return jsonCreateRequestToTransportTransform_8(payload)!;
}export function updatePayloadToTransport_2(payload: FeatureUpdateRequest) {
  return jsonFeatureUpdateRequestToTransportTransform(payload)!;
}export function createCostBasisPayloadToTransport(payload: CreateRequest_5) {
  return jsonCreateRequestToTransportTransform_7(payload)!;
}export function createPayloadToTransport_2(payload: CreateRequest_6) {
  return jsonCreateRequestToTransportTransform_6(payload)!;
}export function createPayloadToTransport_3(payload: CreateRequest_7) {
  return jsonCreateRequestToTransportTransform_5(payload)!;
}export function upsertPayloadToTransport(payload: UpsertRequest_4) {
  return jsonUpsertRequestToTransportTransform_5(payload)!;
}export function createPayloadToTransport_4(payload: CreateRequest_8) {
  return jsonCreateRequestToTransportTransform_4(payload)!;
}export function updatePayloadToTransport_3(payload: UpsertRequest_5) {
  return jsonUpsertRequestToTransportTransform_4(payload)!;
}export function createPayloadToTransport_5(payload: SubscriptionCreate) {
  return jsonSubscriptionCreateToTransportTransform(payload)!;
}export function cancelPayloadToTransport(payload: SubscriptionCancel) {
  return jsonSubscriptionCancelToTransportTransform(payload)!;
}export function changePayloadToTransport(payload: SubscriptionChange) {
  return jsonSubscriptionChangeToTransportTransform(payload)!;
}export function updateExternalSettlementPayloadToTransport(
  payload: UpdateCreditGrantExternalSettlementRequest,
) {
  return jsonUpdateCreditGrantExternalSettlementRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_6(payload: CreateRequest_9) {
  return jsonCreateRequestToTransportTransform_3(payload)!;
}export function createPayloadToTransport_7(payload: CreateRequestNested) {
  return jsonCreateRequestNestedToTransportTransform(payload)!;
}export function upsertPayloadToTransport_2(payload: UpsertRequest_6) {
  return jsonUpsertRequestToTransportTransform_2(payload)!;
}export function upsertAppDataPayloadToTransport(payload: UpsertRequest_7) {
  return jsonUpsertRequestToTransportTransform_3(payload)!;
}export function createCheckoutSessionPayloadToTransport(
  payload: CustomerBillingStripeCreateCheckoutSessionRequest,
) {
  return jsonCustomerBillingStripeCreateCheckoutSessionRequestToTransportTransform(payload)!;
}export function createPortalSessionPayloadToTransport(
  payload: CustomerBillingStripeCreateCustomerPortalSessionRequest,
) {
  return jsonCustomerBillingStripeCreateCustomerPortalSessionRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_8(payload: CreateRequest_10) {
  return jsonCreateRequestToTransportTransform_2(payload)!;
}export function upsertPayloadToTransport_3(payload: UpsertRequest_8) {
  return jsonUpsertRequestToTransportTransform(payload)!;
}export function queryPayloadToTransport_2(payload: MeterQueryRequest) {
  return jsonMeterQueryRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_9(payload: CreateRequest_11) {
  return jsonCreateRequestToTransportTransform(payload)!;
}export function updatePayloadToTransport_4(payload: UpdateRequest_2) {
  return jsonUpdateRequestToTransportTransform(payload)!;
}export function ingestEventPayloadToTransport(payload: MeteringEvent) {
  return jsonMeteringEventToTransportTransform(payload)!;
}export function ingestEventsPayloadToTransport(payload: Array<MeteringEvent>) {
  return jsonArrayMeteringEventToTransportTransform(payload)!;
}export function ingestEventsJsonPayloadToTransport(
  payload: MeteringEvent | Array<MeteringEvent>,
) {
  return payload!;
}export function ingestEventPayloadToTransport_2(payload: MeteringEvent) {
  return jsonMeteringEventToTransportTransform(payload)!;
}export function ingestEventsPayloadToTransport_2(
  payload: Array<MeteringEvent>,
) {
  return jsonArrayMeteringEventToTransportTransform(payload)!;
}export function ingestEventsJsonPayloadToTransport_2(
  payload: MeteringEvent | Array<MeteringEvent>,
) {
  return payload!;
}export function queryPayloadToTransport_3(payload: MeterQueryRequest) {
  return jsonMeterQueryRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_10(payload: CreateRequest_11) {
  return jsonCreateRequestToTransportTransform(payload)!;
}export function updatePayloadToTransport_5(payload: UpdateRequest_2) {
  return jsonUpdateRequestToTransportTransform(payload)!;
}export function updateExternalSettlementPayloadToTransport_2(
  payload: UpdateCreditGrantExternalSettlementRequest,
) {
  return jsonUpdateCreditGrantExternalSettlementRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_11(payload: CreateRequest_9) {
  return jsonCreateRequestToTransportTransform_3(payload)!;
}export function createPayloadToTransport_12(payload: CreateRequestNested) {
  return jsonCreateRequestNestedToTransportTransform(payload)!;
}export function upsertPayloadToTransport_4(payload: UpsertRequest_6) {
  return jsonUpsertRequestToTransportTransform_2(payload)!;
}export function upsertAppDataPayloadToTransport_2(payload: UpsertRequest_7) {
  return jsonUpsertRequestToTransportTransform_3(payload)!;
}export function createCheckoutSessionPayloadToTransport_2(
  payload: CustomerBillingStripeCreateCheckoutSessionRequest,
) {
  return jsonCustomerBillingStripeCreateCheckoutSessionRequestToTransportTransform(payload)!;
}export function createPortalSessionPayloadToTransport_2(
  payload: CustomerBillingStripeCreateCustomerPortalSessionRequest,
) {
  return jsonCustomerBillingStripeCreateCustomerPortalSessionRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_13(payload: CreateRequest_10) {
  return jsonCreateRequestToTransportTransform_2(payload)!;
}export function upsertPayloadToTransport_5(payload: UpsertRequest_8) {
  return jsonUpsertRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_14(payload: CreateRequest_8) {
  return jsonCreateRequestToTransportTransform_4(payload)!;
}export function updatePayloadToTransport_6(payload: UpsertRequest_5) {
  return jsonUpsertRequestToTransportTransform_4(payload)!;
}export function createCostBasisPayloadToTransport_2(payload: CreateRequest_5) {
  return jsonCreateRequestToTransportTransform_7(payload)!;
}export function createPayloadToTransport_15(payload: CreateRequest_6) {
  return jsonCreateRequestToTransportTransform_6(payload)!;
}export function createPlanAddonPayloadToTransport_2(payload: CreateRequest) {
  return jsonCreateRequestToTransportTransform_11(payload)!;
}export function updatePlanAddonPayloadToTransport_2(payload: UpsertRequest) {
  return jsonUpsertRequestToTransportTransform_8(payload)!;
}export function createAddonPayloadToTransport_2(payload: CreateRequest_2) {
  return jsonCreateRequestToTransportTransform_10(payload)!;
}export function updateAddonPayloadToTransport_2(payload: UpsertRequest_2) {
  return jsonUpsertRequestToTransportTransform_7(payload)!;
}export function createPlanPayloadToTransport_2(payload: CreateRequest_3) {
  return jsonCreateRequestToTransportTransform_9(payload)!;
}export function updatePlanPayloadToTransport_2(payload: UpsertRequest_3) {
  return jsonUpsertRequestToTransportTransform_6(payload)!;
}export function queryCostPayloadToTransport_2(payload: MeterQueryRequest) {
  return jsonMeterQueryRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_16(payload: CreateRequest_4) {
  return jsonCreateRequestToTransportTransform_8(payload)!;
}export function updatePayloadToTransport_7(payload: FeatureUpdateRequest) {
  return jsonFeatureUpdateRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_17(payload: CreateRequest_7) {
  return jsonCreateRequestToTransportTransform_5(payload)!;
}export function upsertPayloadToTransport_6(payload: UpsertRequest_4) {
  return jsonUpsertRequestToTransportTransform_5(payload)!;
}export function createPayloadToTransport_18(payload: SubscriptionCreate) {
  return jsonSubscriptionCreateToTransportTransform(payload)!;
}export function cancelPayloadToTransport_2(payload: SubscriptionCancel) {
  return jsonSubscriptionCancelToTransportTransform(payload)!;
}export function changePayloadToTransport_2(payload: SubscriptionChange) {
  return jsonSubscriptionChangeToTransportTransform(payload)!;
}export function createOverridePayloadToTransport_2(payload: OverrideCreate) {
  return jsonOverrideCreateToTransportTransform(payload)!;
}export function updatePayloadToTransport_8(payload: UpdateRequest) {
  return jsonUpdateRequestToTransportTransform_2(payload)!;
}export function queryPayloadToTransport_4(payload: GovernanceQueryRequest) {
  return jsonGovernanceQueryRequestToTransportTransform(payload)!;
}export function queryPayloadToTransport_5(payload: GovernanceQueryRequest) {
  return jsonGovernanceQueryRequestToTransportTransform(payload)!;
}export function updatePayloadToTransport_9(payload: UpdateRequest) {
  return jsonUpdateRequestToTransportTransform_2(payload)!;
}export function createPlanAddonPayloadToTransport_3(payload: CreateRequest) {
  return jsonCreateRequestToTransportTransform_11(payload)!;
}export function updatePlanAddonPayloadToTransport_3(payload: UpsertRequest) {
  return jsonUpsertRequestToTransportTransform_8(payload)!;
}export function createAddonPayloadToTransport_3(payload: CreateRequest_2) {
  return jsonCreateRequestToTransportTransform_10(payload)!;
}export function updateAddonPayloadToTransport_3(payload: UpsertRequest_2) {
  return jsonUpsertRequestToTransportTransform_7(payload)!;
}export function createPlanPayloadToTransport_3(payload: CreateRequest_3) {
  return jsonCreateRequestToTransportTransform_9(payload)!;
}export function updatePlanPayloadToTransport_3(payload: UpsertRequest_3) {
  return jsonUpsertRequestToTransportTransform_6(payload)!;
}export function createOverridePayloadToTransport_3(payload: OverrideCreate) {
  return jsonOverrideCreateToTransportTransform(payload)!;
}export function queryCostPayloadToTransport_3(payload: MeterQueryRequest) {
  return jsonMeterQueryRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_19(payload: CreateRequest_4) {
  return jsonCreateRequestToTransportTransform_8(payload)!;
}export function updatePayloadToTransport_10(payload: FeatureUpdateRequest) {
  return jsonFeatureUpdateRequestToTransportTransform(payload)!;
}export function createCostBasisPayloadToTransport_3(payload: CreateRequest_5) {
  return jsonCreateRequestToTransportTransform_7(payload)!;
}export function createPayloadToTransport_20(payload: CreateRequest_6) {
  return jsonCreateRequestToTransportTransform_6(payload)!;
}export function createPayloadToTransport_21(payload: CreateRequest_7) {
  return jsonCreateRequestToTransportTransform_5(payload)!;
}export function upsertPayloadToTransport_7(payload: UpsertRequest_4) {
  return jsonUpsertRequestToTransportTransform_5(payload)!;
}export function createPayloadToTransport_22(payload: CreateRequest_8) {
  return jsonCreateRequestToTransportTransform_4(payload)!;
}export function updatePayloadToTransport_11(payload: UpsertRequest_5) {
  return jsonUpsertRequestToTransportTransform_4(payload)!;
}export function createPayloadToTransport_23(payload: SubscriptionCreate) {
  return jsonSubscriptionCreateToTransportTransform(payload)!;
}export function cancelPayloadToTransport_3(payload: SubscriptionCancel) {
  return jsonSubscriptionCancelToTransportTransform(payload)!;
}export function changePayloadToTransport_3(payload: SubscriptionChange) {
  return jsonSubscriptionChangeToTransportTransform(payload)!;
}export function updateExternalSettlementPayloadToTransport_3(
  payload: UpdateCreditGrantExternalSettlementRequest,
) {
  return jsonUpdateCreditGrantExternalSettlementRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_24(payload: CreateRequest_9) {
  return jsonCreateRequestToTransportTransform_3(payload)!;
}export function createPayloadToTransport_25(payload: CreateRequestNested) {
  return jsonCreateRequestNestedToTransportTransform(payload)!;
}export function upsertPayloadToTransport_8(payload: UpsertRequest_6) {
  return jsonUpsertRequestToTransportTransform_2(payload)!;
}export function upsertAppDataPayloadToTransport_3(payload: UpsertRequest_7) {
  return jsonUpsertRequestToTransportTransform_3(payload)!;
}export function createCheckoutSessionPayloadToTransport_3(
  payload: CustomerBillingStripeCreateCheckoutSessionRequest,
) {
  return jsonCustomerBillingStripeCreateCheckoutSessionRequestToTransportTransform(payload)!;
}export function createPortalSessionPayloadToTransport_3(
  payload: CustomerBillingStripeCreateCustomerPortalSessionRequest,
) {
  return jsonCustomerBillingStripeCreateCustomerPortalSessionRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_26(payload: CreateRequest_10) {
  return jsonCreateRequestToTransportTransform_2(payload)!;
}export function upsertPayloadToTransport_9(payload: UpsertRequest_8) {
  return jsonUpsertRequestToTransportTransform(payload)!;
}export function queryPayloadToTransport_6(payload: MeterQueryRequest) {
  return jsonMeterQueryRequestToTransportTransform(payload)!;
}export function createPayloadToTransport_27(payload: CreateRequest_11) {
  return jsonCreateRequestToTransportTransform(payload)!;
}export function updatePayloadToTransport_12(payload: UpdateRequest_2) {
  return jsonUpdateRequestToTransportTransform(payload)!;
}export function ingestEventPayloadToTransport_3(payload: MeteringEvent) {
  return jsonMeteringEventToTransportTransform(payload)!;
}export function ingestEventsPayloadToTransport_3(
  payload: Array<MeteringEvent>,
) {
  return jsonArrayMeteringEventToTransportTransform(payload)!;
}export function ingestEventsJsonPayloadToTransport_3(
  payload: MeteringEvent | Array<MeteringEvent>,
) {
  return payload!;
}export function jsonCursorPaginationQueryPageToTransportTransform(
  input_?: CursorPaginationQueryPage | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    size: input_.size,after: input_.after,before: input_.before
  }!;
}export function jsonCursorPaginationQueryPageToApplicationTransform(
  input_?: any,
): CursorPaginationQueryPage {
  if(!input_) {
    return input_ as any;
  }
    return {
    size: input_.size,after: input_.after,before: input_.before
  }!;
}export function jsonListEventsParamsFilterToTransportTransform(
  input_?: ListEventsParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: jsonStringFieldFilterToTransportTransform(input_.id),source: jsonStringFieldFilterToTransportTransform(input_.source),subject: jsonStringFieldFilterToTransportTransform(input_.subject),type: jsonStringFieldFilterToTransportTransform(input_.type),customer_id: jsonUlidFieldFilterToTransportTransform(input_.customerId),time: jsonDateTimeFieldFilterToTransportTransform(input_.time),ingested_at: jsonDateTimeFieldFilterToTransportTransform(input_.ingestedAt),stored_at: jsonDateTimeFieldFilterToTransportTransform(input_.storedAt)
  }!;
}export function jsonListEventsParamsFilterToApplicationTransform(
  input_?: any,
): ListEventsParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: jsonStringFieldFilterToApplicationTransform(input_.id),source: jsonStringFieldFilterToApplicationTransform(input_.source),subject: jsonStringFieldFilterToApplicationTransform(input_.subject),type: jsonStringFieldFilterToApplicationTransform(input_.type),customerId: jsonUlidFieldFilterToApplicationTransform(input_.customer_id),time: jsonDateTimeFieldFilterToApplicationTransform(input_.time),ingestedAt: jsonDateTimeFieldFilterToApplicationTransform(input_.ingested_at),storedAt: jsonDateTimeFieldFilterToApplicationTransform(input_.stored_at)
  }!;
}export function jsonStringFieldFilterToTransportTransform(
  input_?: StringFieldFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonStringFieldFilterToApplicationTransform(
  input_?: any,
): StringFieldFilter {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonArrayStringToTransportTransform(
  items_?: Array<string> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayStringToApplicationTransform(
  items_?: any,
): Array<string> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonUlidFieldFilterToTransportTransform(
  input_?: UlidFieldFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonUlidFieldFilterToApplicationTransform(
  input_?: any,
): UlidFieldFilter {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonArrayUlidToTransportTransform(
  items_?: Array<string> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayUlidToApplicationTransform(
  items_?: any,
): Array<string> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonDateTimeFieldFilterToTransportTransform(
  input_?: DateTimeFieldFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonDateTimeFieldFilterToApplicationTransform(
  input_?: any,
): DateTimeFieldFilter {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonSortQueryToTransportTransform(
  input_?: SortQuery | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {}!;
}export function jsonSortQueryToApplicationTransform(input_?: any): SortQuery {
  if(!input_) {
    return input_ as any;
  }
    return {}!;
}export function jsonArrayIngestedEventToTransportTransform(
  items_?: Array<IngestedEvent> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonIngestedEventToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayIngestedEventToApplicationTransform(
  items_?: any,
): Array<IngestedEvent> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonIngestedEventToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonIngestedEventToTransportTransform(
  input_?: IngestedEvent | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    event: jsonMeteringEventToTransportTransform(input_.event),customer: jsonResourceReferenceToTransportTransform(input_.customer),ingested_at: dateRfc3339Serializer(input_.ingestedAt),stored_at: dateRfc3339Serializer(input_.storedAt),validation_errors: jsonArrayIngestedEventValidationErrorToTransportTransform(input_.validationErrors)
  }!;
}export function jsonIngestedEventToApplicationTransform(
  input_?: any,
): IngestedEvent {
  if(!input_) {
    return input_ as any;
  }
    return {
    event: jsonMeteringEventToApplicationTransform(input_.event),customer: jsonResourceReferenceToApplicationTransform(input_.customer),ingestedAt: dateDeserializer(input_.ingested_at)!,storedAt: dateDeserializer(input_.stored_at)!,validationErrors: jsonArrayIngestedEventValidationErrorToApplicationTransform(input_.validation_errors)
  }!;
}export function jsonMeteringEventToTransportTransform(
  input_?: MeteringEvent | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,source: input_.source,specversion: input_.specversion,type: input_.type,datacontenttype: input_.datacontenttype,dataschema: input_.dataschema,subject: input_.subject,time: dateRfc3339Serializer(input_.time),data: jsonRecordUnknownToTransportTransform(input_.data)
  }!;
}export function jsonMeteringEventToApplicationTransform(
  input_?: any,
): MeteringEvent {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,source: input_.source,specversion: input_.specversion,type: input_.type,datacontenttype: input_.datacontenttype,dataschema: input_.dataschema,subject: input_.subject,time: dateDeserializer(input_.time)!,data: jsonRecordUnknownToApplicationTransform(input_.data)
  }!;
}export function jsonRecordUnknownToTransportTransform(
  items_?: Record<string, any> | null,
): any {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = value as any;
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonRecordUnknownToApplicationTransform(
  items_?: any,
): Record<string, any> {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = value as any;
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonResourceReferenceToTransportTransform(
  input_?: ResourceReference | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonResourceReferenceToApplicationTransform(
  input_?: any,
): ResourceReference {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonArrayIngestedEventValidationErrorToTransportTransform(
  items_?: Array<IngestedEventValidationError> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonIngestedEventValidationErrorToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayIngestedEventValidationErrorToApplicationTransform(
  items_?: any,
): Array<IngestedEventValidationError> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonIngestedEventValidationErrorToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonIngestedEventValidationErrorToTransportTransform(
  input_?: IngestedEventValidationError | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code,message: input_.message,attributes: jsonRecordUnknownToTransportTransform(input_.attributes)
  }!;
}export function jsonIngestedEventValidationErrorToApplicationTransform(
  input_?: any,
): IngestedEventValidationError {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code,message: input_.message,attributes: jsonRecordUnknownToApplicationTransform(input_.attributes)
  }!;
}export function jsonCursorMetaToTransportTransform(
  input_?: CursorMeta | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    page: jsonCursorMetaPageToTransportTransform(input_.page)
  }!;
}export function jsonCursorMetaToApplicationTransform(
  input_?: any,
): CursorMeta {
  if(!input_) {
    return input_ as any;
  }
    return {
    page: jsonCursorMetaPageToApplicationTransform(input_.page)
  }!;
}export function jsonCursorMetaPageToTransportTransform(
  input_?: CursorMetaPage | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    first: input_.first,last: input_.last,next: input_.next,previous: input_.previous,size: input_.size
  }!;
}export function jsonCursorMetaPageToApplicationTransform(
  input_?: any,
): CursorMetaPage {
  if(!input_) {
    return input_ as any;
  }
    return {
    first: input_.first,last: input_.last,next: input_.next,previous: input_.previous,size: input_.size
  }!;
}export function jsonArrayMeteringEventToTransportTransform(
  items_?: Array<MeteringEvent> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonMeteringEventToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayMeteringEventToApplicationTransform(
  items_?: any,
): Array<MeteringEvent> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonMeteringEventToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonCreateRequestToTransportTransform(
  input_?: CreateRequest_11 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),key: input_.key,aggregation: jsonMeterAggregationToTransportTransform(input_.aggregation),event_type: input_.eventType,events_from: dateRfc3339Serializer(input_.eventsFrom),value_property: input_.valueProperty,dimensions: jsonRecordStringToTransportTransform(input_.dimensions)
  }!;
}export function jsonCreateRequestToApplicationTransform(
  input_?: any,
): CreateRequest_11 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),key: input_.key,aggregation: jsonMeterAggregationToApplicationTransform(input_.aggregation),eventType: input_.event_type,eventsFrom: dateDeserializer(input_.events_from)!,valueProperty: input_.value_property,dimensions: jsonRecordStringToApplicationTransform(input_.dimensions)
  }!;
}export function jsonLabelsToTransportTransform(input_?: Labels | null): any {
  if(!input_) {
    return input_ as any;
  }
    return {}!;
}export function jsonLabelsToApplicationTransform(input_?: any): Labels {
  if(!input_) {
    return input_ as any;
  }
    return {}!;
}export function jsonMeterAggregationToTransportTransform(
  input_?: MeterAggregation | null,
): any {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonMeterAggregationToApplicationTransform(
  input_?: any,
): MeterAggregation {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonRecordStringToTransportTransform(
  items_?: Record<string, any> | null,
): any {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = value as any;
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonRecordStringToApplicationTransform(
  items_?: any,
): Record<string, any> {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = value as any;
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonMeterToTransportTransform(input_?: Meter | null): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),key: input_.key,aggregation: jsonMeterAggregationToTransportTransform(input_.aggregation),event_type: input_.eventType,events_from: dateRfc3339Serializer(input_.eventsFrom),value_property: input_.valueProperty,dimensions: jsonRecordStringToTransportTransform(input_.dimensions)
  }!;
}export function jsonMeterToApplicationTransform(input_?: any): Meter {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,key: input_.key,aggregation: jsonMeterAggregationToApplicationTransform(input_.aggregation),eventType: input_.event_type,eventsFrom: dateDeserializer(input_.events_from)!,valueProperty: input_.value_property,dimensions: jsonRecordStringToApplicationTransform(input_.dimensions)
  }!;
}export function jsonListMetersParamsFilterToTransportTransform(
  input_?: ListMetersParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    key: jsonStringFieldFilterToTransportTransform(input_.key),name: jsonStringFieldFilterToTransportTransform(input_.name)
  }!;
}export function jsonListMetersParamsFilterToApplicationTransform(
  input_?: any,
): ListMetersParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    key: jsonStringFieldFilterToApplicationTransform(input_.key),name: jsonStringFieldFilterToApplicationTransform(input_.name)
  }!;
}export function jsonArrayMeterToTransportTransform(
  items_?: Array<Meter> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonMeterToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayMeterToApplicationTransform(
  items_?: any,
): Array<Meter> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonMeterToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonPageMetaToTransportTransform(
  input_?: PageMeta | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    page: jsonPagePaginatedMetaToTransportTransform(input_.page)
  }!;
}export function jsonPageMetaToApplicationTransform(input_?: any): PageMeta {
  if(!input_) {
    return input_ as any;
  }
    return {
    page: jsonPagePaginatedMetaToApplicationTransform(input_.page)
  }!;
}export function jsonPagePaginatedMetaToTransportTransform(
  input_?: PagePaginatedMeta | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    number: input_.number,size: input_.size,total: input_.total
  }!;
}export function jsonPagePaginatedMetaToApplicationTransform(
  input_?: any,
): PagePaginatedMeta {
  if(!input_) {
    return input_ as any;
  }
    return {
    number: input_.number,size: input_.size,total: input_.total
  }!;
}export function jsonUpdateRequestToTransportTransform(
  input_?: UpdateRequest_2 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),dimensions: jsonRecordStringToTransportTransform(input_.dimensions)
  }!;
}export function jsonUpdateRequestToApplicationTransform(
  input_?: any,
): UpdateRequest_2 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),dimensions: jsonRecordStringToApplicationTransform(input_.dimensions)
  }!;
}export function jsonMeterQueryRequestToTransportTransform(
  input_?: MeterQueryRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    from: dateRfc3339Serializer(input_.from),to: dateRfc3339Serializer(input_.to),granularity: jsonMeterQueryGranularityToTransportTransform(input_.granularity),time_zone: input_.timeZone,group_by_dimensions: jsonArrayStringToTransportTransform(input_.groupByDimensions),filters: jsonMeterQueryFiltersToTransportTransform(input_.filters)
  }!;
}export function jsonMeterQueryRequestToApplicationTransform(
  input_?: any,
): MeterQueryRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    from: dateDeserializer(input_.from)!,to: dateDeserializer(input_.to)!,granularity: jsonMeterQueryGranularityToApplicationTransform(input_.granularity),timeZone: input_.time_zone,groupByDimensions: jsonArrayStringToApplicationTransform(input_.group_by_dimensions),filters: jsonMeterQueryFiltersToApplicationTransform(input_.filters)
  }!;
}export function jsonMeterQueryGranularityToTransportTransform(
  input_?: MeterQueryGranularity | null,
): any {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonMeterQueryGranularityToApplicationTransform(
  input_?: any,
): MeterQueryGranularity {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonMeterQueryFiltersToTransportTransform(
  input_?: MeterQueryFilters | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    dimensions: jsonRecordQueryFilterStringMapItemToTransportTransform(input_.dimensions)
  }!;
}export function jsonMeterQueryFiltersToApplicationTransform(
  input_?: any,
): MeterQueryFilters {
  if(!input_) {
    return input_ as any;
  }
    return {
    dimensions: jsonRecordQueryFilterStringMapItemToApplicationTransform(input_.dimensions)
  }!;
}export function jsonRecordQueryFilterStringMapItemToTransportTransform(
  items_?: Record<string, any> | null,
): any {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = jsonQueryFilterStringMapItemToTransportTransform(value as any);
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonRecordQueryFilterStringMapItemToApplicationTransform(
  items_?: any,
): Record<string, any> {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = jsonQueryFilterStringMapItemToApplicationTransform(value as any);
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonQueryFilterStringMapItemToTransportTransform(
  input_?: QueryFilterStringMapItem | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    exists: input_.exists,eq: input_.eq,neq: input_.neq,in: jsonArrayStringToTransportTransform(input_.in_),nin: jsonArrayStringToTransportTransform(input_.nin),contains: input_.contains,ncontains: input_.ncontains,and: jsonArrayQueryFilterStringToTransportTransform(input_.and),or: jsonArrayQueryFilterStringToTransportTransform(input_.or)
  }!;
}export function jsonQueryFilterStringMapItemToApplicationTransform(
  input_?: any,
): QueryFilterStringMapItem {
  if(!input_) {
    return input_ as any;
  }
    return {
    exists: input_.exists,eq: input_.eq,neq: input_.neq,in_: jsonArrayStringToApplicationTransform(input_.in),nin: jsonArrayStringToApplicationTransform(input_.nin),contains: input_.contains,ncontains: input_.ncontains,and: jsonArrayQueryFilterStringToApplicationTransform(input_.and),or: jsonArrayQueryFilterStringToApplicationTransform(input_.or)
  }!;
}export function jsonArrayQueryFilterStringToTransportTransform(
  items_?: Array<QueryFilterString> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonQueryFilterStringToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayQueryFilterStringToApplicationTransform(
  items_?: any,
): Array<QueryFilterString> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonQueryFilterStringToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonQueryFilterStringToTransportTransform(
  input_?: QueryFilterString | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    eq: input_.eq,neq: input_.neq,in: jsonArrayStringToTransportTransform(input_.in_),nin: jsonArrayStringToTransportTransform(input_.nin),contains: input_.contains,ncontains: input_.ncontains,and: jsonArrayQueryFilterStringToTransportTransform(input_.and),or: jsonArrayQueryFilterStringToTransportTransform(input_.or)
  }!;
}export function jsonQueryFilterStringToApplicationTransform(
  input_?: any,
): QueryFilterString {
  if(!input_) {
    return input_ as any;
  }
    return {
    eq: input_.eq,neq: input_.neq,in_: jsonArrayStringToApplicationTransform(input_.in),nin: jsonArrayStringToApplicationTransform(input_.nin),contains: input_.contains,ncontains: input_.ncontains,and: jsonArrayQueryFilterStringToApplicationTransform(input_.and),or: jsonArrayQueryFilterStringToApplicationTransform(input_.or)
  }!;
}export function jsonMeterQueryResultToTransportTransform(
  input_?: MeterQueryResult | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    from: dateRfc3339Serializer(input_.from),to: dateRfc3339Serializer(input_.to),data: jsonArrayMeterQueryRowToTransportTransform(input_.data)
  }!;
}export function jsonMeterQueryResultToApplicationTransform(
  input_?: any,
): MeterQueryResult {
  if(!input_) {
    return input_ as any;
  }
    return {
    from: dateDeserializer(input_.from)!,to: dateDeserializer(input_.to)!,data: jsonArrayMeterQueryRowToApplicationTransform(input_.data)
  }!;
}export function jsonArrayMeterQueryRowToTransportTransform(
  items_?: Array<MeterQueryRow> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonMeterQueryRowToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayMeterQueryRowToApplicationTransform(
  items_?: any,
): Array<MeterQueryRow> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonMeterQueryRowToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonMeterQueryRowToTransportTransform(
  input_?: MeterQueryRow | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    value: input_.value,from: dateRfc3339Serializer(input_.from),to: dateRfc3339Serializer(input_.to),dimensions: jsonRecordStringToTransportTransform(input_.dimensions)
  }!;
}export function jsonMeterQueryRowToApplicationTransform(
  input_?: any,
): MeterQueryRow {
  if(!input_) {
    return input_ as any;
  }
    return {
    value: input_.value,from: dateDeserializer(input_.from)!,to: dateDeserializer(input_.to)!,dimensions: jsonRecordStringToApplicationTransform(input_.dimensions)
  }!;
}export function jsonCreateRequestToTransportTransform_2(
  input_?: CreateRequest_10 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),key: input_.key,usage_attribution: jsonCustomerUsageAttributionToTransportTransform(input_.usageAttribution),primary_email: input_.primaryEmail,currency: input_.currency,billing_address: jsonAddressToTransportTransform(input_.billingAddress)
  }!;
}export function jsonCreateRequestToApplicationTransform_2(
  input_?: any,
): CreateRequest_10 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),key: input_.key,usageAttribution: jsonCustomerUsageAttributionToApplicationTransform(input_.usage_attribution),primaryEmail: input_.primary_email,currency: input_.currency,billingAddress: jsonAddressToApplicationTransform(input_.billing_address)
  }!;
}export function jsonCustomerUsageAttributionToTransportTransform(
  input_?: CustomerUsageAttribution | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    subject_keys: jsonArrayUsageAttributionSubjectKeyToTransportTransform(input_.subjectKeys)
  }!;
}export function jsonCustomerUsageAttributionToApplicationTransform(
  input_?: any,
): CustomerUsageAttribution {
  if(!input_) {
    return input_ as any;
  }
    return {
    subjectKeys: jsonArrayUsageAttributionSubjectKeyToApplicationTransform(input_.subject_keys)
  }!;
}export function jsonArrayUsageAttributionSubjectKeyToTransportTransform(
  items_?: Array<string> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayUsageAttributionSubjectKeyToApplicationTransform(
  items_?: any,
): Array<string> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonAddressToTransportTransform(input_?: Address | null): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    country: input_.country,postal_code: input_.postalCode,state: input_.state,city: input_.city,line1: input_.line1,line2: input_.line2,phone_number: input_.phoneNumber
  }!;
}export function jsonAddressToApplicationTransform(input_?: any): Address {
  if(!input_) {
    return input_ as any;
  }
    return {
    country: input_.country,postalCode: input_.postal_code,state: input_.state,city: input_.city,line1: input_.line1,line2: input_.line2,phoneNumber: input_.phone_number
  }!;
}export function jsonCustomerToTransportTransform(
  input_?: Customer | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),key: input_.key,usage_attribution: jsonCustomerUsageAttributionToTransportTransform(input_.usageAttribution),primary_email: input_.primaryEmail,currency: input_.currency,billing_address: jsonAddressToTransportTransform(input_.billingAddress)
  }!;
}export function jsonCustomerToApplicationTransform(input_?: any): Customer {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,key: input_.key,usageAttribution: jsonCustomerUsageAttributionToApplicationTransform(input_.usage_attribution),primaryEmail: input_.primary_email,currency: input_.currency,billingAddress: jsonAddressToApplicationTransform(input_.billing_address)
  }!;
}export function jsonListCustomersParamsFilterToTransportTransform(
  input_?: ListCustomersParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    key: jsonStringFieldFilterToTransportTransform(input_.key),name: jsonStringFieldFilterToTransportTransform(input_.name),primary_email: jsonStringFieldFilterToTransportTransform(input_.primaryEmail),usage_attribution_subject_key: jsonStringFieldFilterToTransportTransform(input_.usageAttributionSubjectKey),plan_key: jsonStringFieldFilterToTransportTransform(input_.planKey),billing_profile_id: jsonUlidFieldFilterToTransportTransform(input_.billingProfileId)
  }!;
}export function jsonListCustomersParamsFilterToApplicationTransform(
  input_?: any,
): ListCustomersParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    key: jsonStringFieldFilterToApplicationTransform(input_.key),name: jsonStringFieldFilterToApplicationTransform(input_.name),primaryEmail: jsonStringFieldFilterToApplicationTransform(input_.primary_email),usageAttributionSubjectKey: jsonStringFieldFilterToApplicationTransform(input_.usage_attribution_subject_key),planKey: jsonStringFieldFilterToApplicationTransform(input_.plan_key),billingProfileId: jsonUlidFieldFilterToApplicationTransform(input_.billing_profile_id)
  }!;
}export function jsonArrayCustomerToTransportTransform(
  items_?: Array<Customer> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonCustomerToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayCustomerToApplicationTransform(
  items_?: any,
): Array<Customer> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonCustomerToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonUpsertRequestToTransportTransform(
  input_?: UpsertRequest_8 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),usage_attribution: jsonCustomerUsageAttributionToTransportTransform(input_.usageAttribution),primary_email: input_.primaryEmail,currency: input_.currency,billing_address: jsonAddressToTransportTransform(input_.billingAddress)
  }!;
}export function jsonUpsertRequestToApplicationTransform(
  input_?: any,
): UpsertRequest_8 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),usageAttribution: jsonCustomerUsageAttributionToApplicationTransform(input_.usage_attribution),primaryEmail: input_.primary_email,currency: input_.currency,billingAddress: jsonAddressToApplicationTransform(input_.billing_address)
  }!;
}export function jsonCustomerBillingDataToTransportTransform(
  input_?: CustomerBillingData | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    billing_profile: jsonBillingProfileReferenceToTransportTransform(input_.billingProfile),app_data: jsonAppCustomerDataToTransportTransform(input_.appData)
  }!;
}export function jsonCustomerBillingDataToApplicationTransform(
  input_?: any,
): CustomerBillingData {
  if(!input_) {
    return input_ as any;
  }
    return {
    billingProfile: jsonBillingProfileReferenceToApplicationTransform(input_.billing_profile),appData: jsonAppCustomerDataToApplicationTransform(input_.app_data)
  }!;
}export function jsonBillingProfileReferenceToTransportTransform(
  input_?: BillingProfileReference | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonBillingProfileReferenceToApplicationTransform(
  input_?: any,
): BillingProfileReference {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonAppCustomerDataToTransportTransform(
  input_?: AppCustomerData | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    stripe: jsonAppCustomerDataStripeToTransportTransform(input_.stripe),external_invoicing: jsonAppCustomerDataExternalInvoicingToTransportTransform(input_.externalInvoicing)
  }!;
}export function jsonAppCustomerDataToApplicationTransform(
  input_?: any,
): AppCustomerData {
  if(!input_) {
    return input_ as any;
  }
    return {
    stripe: jsonAppCustomerDataStripeToApplicationTransform(input_.stripe),externalInvoicing: jsonAppCustomerDataExternalInvoicingToApplicationTransform(input_.external_invoicing)
  }!;
}export function jsonAppCustomerDataStripeToTransportTransform(
  input_?: AppCustomerDataStripe | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    customer_id: input_.customerId,default_payment_method_id: input_.defaultPaymentMethodId,labels: jsonLabelsToTransportTransform(input_.labels)
  }!;
}export function jsonAppCustomerDataStripeToApplicationTransform(
  input_?: any,
): AppCustomerDataStripe {
  if(!input_) {
    return input_ as any;
  }
    return {
    customerId: input_.customer_id,defaultPaymentMethodId: input_.default_payment_method_id,labels: jsonLabelsToApplicationTransform(input_.labels)
  }!;
}export function jsonAppCustomerDataExternalInvoicingToTransportTransform(
  input_?: AppCustomerDataExternalInvoicing | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    labels: jsonLabelsToTransportTransform(input_.labels)
  }!;
}export function jsonAppCustomerDataExternalInvoicingToApplicationTransform(
  input_?: any,
): AppCustomerDataExternalInvoicing {
  if(!input_) {
    return input_ as any;
  }
    return {
    labels: jsonLabelsToApplicationTransform(input_.labels)
  }!;
}export function jsonUpsertRequestToTransportTransform_2(
  input_?: UpsertRequest_6 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    billing_profile: jsonBillingProfileReferenceToTransportTransform(input_.billingProfile),app_data: jsonAppCustomerDataToTransportTransform(input_.appData)
  }!;
}export function jsonUpsertRequestToApplicationTransform_2(
  input_?: any,
): UpsertRequest_6 {
  if(!input_) {
    return input_ as any;
  }
    return {
    billingProfile: jsonBillingProfileReferenceToApplicationTransform(input_.billing_profile),appData: jsonAppCustomerDataToApplicationTransform(input_.app_data)
  }!;
}export function jsonUpsertRequestToTransportTransform_3(
  input_?: UpsertRequest_7 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    stripe: jsonAppCustomerDataStripeToTransportTransform(input_.stripe),external_invoicing: jsonAppCustomerDataExternalInvoicingToTransportTransform(input_.externalInvoicing)
  }!;
}export function jsonUpsertRequestToApplicationTransform_3(
  input_?: any,
): UpsertRequest_7 {
  if(!input_) {
    return input_ as any;
  }
    return {
    stripe: jsonAppCustomerDataStripeToApplicationTransform(input_.stripe),externalInvoicing: jsonAppCustomerDataExternalInvoicingToApplicationTransform(input_.external_invoicing)
  }!;
}export function jsonCustomerBillingStripeCreateCheckoutSessionRequestToTransportTransform(
  input_?: CustomerBillingStripeCreateCheckoutSessionRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    stripe_options: jsonCreateStripeCheckoutSessionRequestOptionsToTransportTransform(input_.stripeOptions)
  }!;
}export function jsonCustomerBillingStripeCreateCheckoutSessionRequestToApplicationTransform(
  input_?: any,
): CustomerBillingStripeCreateCheckoutSessionRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    stripeOptions: jsonCreateStripeCheckoutSessionRequestOptionsToApplicationTransform(input_.stripe_options)
  }!;
}export function jsonCreateStripeCheckoutSessionRequestOptionsToTransportTransform(
  input_?: CreateStripeCheckoutSessionRequestOptions | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    billing_address_collection: input_.billingAddressCollection,cancel_url: input_.cancelUrl,client_reference_id: input_.clientReferenceId,customer_update: jsonCreateStripeCheckoutSessionCustomerUpdateToTransportTransform(input_.customerUpdate),consent_collection: jsonCreateStripeCheckoutSessionConsentCollectionToTransportTransform(input_.consentCollection),currency: input_.currency,custom_text: jsonCheckoutSessionCustomTextParamsToTransportTransform(input_.customText),expires_at: input_.expiresAt,locale: input_.locale,metadata: jsonRecordStringToTransportTransform(input_.metadata),return_url: input_.returnUrl,success_url: input_.successUrl,ui_mode: input_.uiMode,payment_method_types: jsonArrayStringToTransportTransform(input_.paymentMethodTypes),redirect_on_completion: input_.redirectOnCompletion,tax_id_collection: jsonCreateCheckoutSessionTaxIdCollectionToTransportTransform(input_.taxIdCollection)
  }!;
}export function jsonCreateStripeCheckoutSessionRequestOptionsToApplicationTransform(
  input_?: any,
): CreateStripeCheckoutSessionRequestOptions {
  if(!input_) {
    return input_ as any;
  }
    return {
    billingAddressCollection: input_.billing_address_collection,cancelUrl: input_.cancel_url,clientReferenceId: input_.client_reference_id,customerUpdate: jsonCreateStripeCheckoutSessionCustomerUpdateToApplicationTransform(input_.customer_update),consentCollection: jsonCreateStripeCheckoutSessionConsentCollectionToApplicationTransform(input_.consent_collection),currency: input_.currency,customText: jsonCheckoutSessionCustomTextParamsToApplicationTransform(input_.custom_text),expiresAt: input_.expires_at,locale: input_.locale,metadata: jsonRecordStringToApplicationTransform(input_.metadata),returnUrl: input_.return_url,successUrl: input_.success_url,uiMode: input_.ui_mode,paymentMethodTypes: jsonArrayStringToApplicationTransform(input_.payment_method_types),redirectOnCompletion: input_.redirect_on_completion,taxIdCollection: jsonCreateCheckoutSessionTaxIdCollectionToApplicationTransform(input_.tax_id_collection)
  }!;
}export function jsonCreateStripeCheckoutSessionCustomerUpdateToTransportTransform(
  input_?: CreateStripeCheckoutSessionCustomerUpdate | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    address: input_.address,name: input_.name,shipping: input_.shipping
  }!;
}export function jsonCreateStripeCheckoutSessionCustomerUpdateToApplicationTransform(
  input_?: any,
): CreateStripeCheckoutSessionCustomerUpdate {
  if(!input_) {
    return input_ as any;
  }
    return {
    address: input_.address,name: input_.name,shipping: input_.shipping
  }!;
}export function jsonCreateStripeCheckoutSessionConsentCollectionToTransportTransform(
  input_?: CreateStripeCheckoutSessionConsentCollection | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    payment_method_reuse_agreement: jsonCreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementToTransportTransform(input_.paymentMethodReuseAgreement),promotions: input_.promotions,terms_of_service: input_.termsOfService
  }!;
}export function jsonCreateStripeCheckoutSessionConsentCollectionToApplicationTransform(
  input_?: any,
): CreateStripeCheckoutSessionConsentCollection {
  if(!input_) {
    return input_ as any;
  }
    return {
    paymentMethodReuseAgreement: jsonCreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementToApplicationTransform(input_.payment_method_reuse_agreement),promotions: input_.promotions,termsOfService: input_.terms_of_service
  }!;
}export function jsonCreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementToTransportTransform(
  input_?: CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    position: input_.position
  }!;
}export function jsonCreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreementToApplicationTransform(
  input_?: any,
): CreateStripeCheckoutSessionConsentCollectionPaymentMethodReuseAgreement {
  if(!input_) {
    return input_ as any;
  }
    return {
    position: input_.position
  }!;
}export function jsonCheckoutSessionCustomTextParamsToTransportTransform(
  input_?: CheckoutSessionCustomTextParams | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    after_submit: {
      message: input_.afterSubmit.message
    },shipping_address: {
      message: input_.shippingAddress.message
    },submit: {
      message: input_.submit.message
    },terms_of_service_acceptance: {
      message: input_.termsOfServiceAcceptance.message
    }
  }!;
}export function jsonCheckoutSessionCustomTextParamsToApplicationTransform(
  input_?: any,
): CheckoutSessionCustomTextParams {
  if(!input_) {
    return input_ as any;
  }
    return {
    afterSubmit: {
      message: input_.after_submit.message
    },shippingAddress: {
      message: input_.shipping_address.message
    },submit: {
      message: input_.submit.message
    },termsOfServiceAcceptance: {
      message: input_.terms_of_service_acceptance.message
    }
  }!;
}export function jsonCreateCheckoutSessionTaxIdCollectionToTransportTransform(
  input_?: CreateCheckoutSessionTaxIdCollection | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    enabled: input_.enabled,required: input_.required
  }!;
}export function jsonCreateCheckoutSessionTaxIdCollectionToApplicationTransform(
  input_?: any,
): CreateCheckoutSessionTaxIdCollection {
  if(!input_) {
    return input_ as any;
  }
    return {
    enabled: input_.enabled,required: input_.required
  }!;
}export function jsonCreateStripeCheckoutSessionResultToTransportTransform(
  input_?: CreateStripeCheckoutSessionResult | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    customer_id: input_.customerId,stripe_customer_id: input_.stripeCustomerId,session_id: input_.sessionId,setup_intent_id: input_.setupIntentId,client_secret: input_.clientSecret,client_reference_id: input_.clientReferenceId,customer_email: input_.customerEmail,currency: input_.currency,created_at: dateRfc3339Serializer(input_.createdAt),expires_at: dateRfc3339Serializer(input_.expiresAt),metadata: jsonRecordStringToTransportTransform(input_.metadata),status: input_.status,url: input_.url,mode: input_.mode,cancel_url: input_.cancelUrl,success_url: input_.successUrl,return_url: input_.returnUrl
  }!;
}export function jsonCreateStripeCheckoutSessionResultToApplicationTransform(
  input_?: any,
): CreateStripeCheckoutSessionResult {
  if(!input_) {
    return input_ as any;
  }
    return {
    customerId: input_.customer_id,stripeCustomerId: input_.stripe_customer_id,sessionId: input_.session_id,setupIntentId: input_.setup_intent_id,clientSecret: input_.client_secret,clientReferenceId: input_.client_reference_id,customerEmail: input_.customer_email,currency: input_.currency,createdAt: dateDeserializer(input_.created_at)!,expiresAt: dateDeserializer(input_.expires_at)!,metadata: jsonRecordStringToApplicationTransform(input_.metadata),status: input_.status,url: input_.url,mode: input_.mode,cancelUrl: input_.cancel_url,successUrl: input_.success_url,returnUrl: input_.return_url
  }!;
}export function jsonCustomerBillingStripeCreateCustomerPortalSessionRequestToTransportTransform(
  input_?: CustomerBillingStripeCreateCustomerPortalSessionRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    stripe_options: jsonCreateStripeCustomerPortalSessionOptionsToTransportTransform(input_.stripeOptions)
  }!;
}export function jsonCustomerBillingStripeCreateCustomerPortalSessionRequestToApplicationTransform(
  input_?: any,
): CustomerBillingStripeCreateCustomerPortalSessionRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    stripeOptions: jsonCreateStripeCustomerPortalSessionOptionsToApplicationTransform(input_.stripe_options)
  }!;
}export function jsonCreateStripeCustomerPortalSessionOptionsToTransportTransform(
  input_?: CreateStripeCustomerPortalSessionOptions | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    configuration_id: input_.configurationId,locale: input_.locale,return_url: input_.returnUrl
  }!;
}export function jsonCreateStripeCustomerPortalSessionOptionsToApplicationTransform(
  input_?: any,
): CreateStripeCustomerPortalSessionOptions {
  if(!input_) {
    return input_ as any;
  }
    return {
    configurationId: input_.configuration_id,locale: input_.locale,returnUrl: input_.return_url
  }!;
}export function jsonCreateStripeCustomerPortalSessionResultToTransportTransform(
  input_?: CreateStripeCustomerPortalSessionResult | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,stripe_customer_id: input_.stripeCustomerId,configuration_id: input_.configurationId,livemode: input_.livemode,created_at: dateRfc3339Serializer(input_.createdAt),return_url: input_.returnUrl,locale: input_.locale,url: input_.url
  }!;
}export function jsonCreateStripeCustomerPortalSessionResultToApplicationTransform(
  input_?: any,
): CreateStripeCustomerPortalSessionResult {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,stripeCustomerId: input_.stripe_customer_id,configurationId: input_.configuration_id,livemode: input_.livemode,createdAt: dateDeserializer(input_.created_at)!,returnUrl: input_.return_url,locale: input_.locale,url: input_.url
  }!;
}export function jsonListCustomerEntitlementAccessResponseDataToTransportTransform(
  input_?: ListCustomerEntitlementAccessResponseData | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    data: jsonArrayEntitlementAccessResultToTransportTransform(input_.data)
  }!;
}export function jsonListCustomerEntitlementAccessResponseDataToApplicationTransform(
  input_?: any,
): ListCustomerEntitlementAccessResponseData {
  if(!input_) {
    return input_ as any;
  }
    return {
    data: jsonArrayEntitlementAccessResultToApplicationTransform(input_.data)
  }!;
}export function jsonArrayEntitlementAccessResultToTransportTransform(
  items_?: Array<EntitlementAccessResult> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonEntitlementAccessResultToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayEntitlementAccessResultToApplicationTransform(
  items_?: any,
): Array<EntitlementAccessResult> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonEntitlementAccessResultToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonEntitlementAccessResultToTransportTransform(
  input_?: EntitlementAccessResult | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,feature_key: input_.featureKey,has_access: input_.hasAccess,config: input_.config
  }!;
}export function jsonEntitlementAccessResultToApplicationTransform(
  input_?: any,
): EntitlementAccessResult {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,featureKey: input_.feature_key,hasAccess: input_.has_access,config: input_.config
  }!;
}export function jsonCreateRequestNestedToTransportTransform(
  input_?: CreateRequestNested | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),funding_method: input_.fundingMethod,currency: jsonCurrencyCodeToTransportTransform(input_.currency),amount: input_.amount,purchase: {
      currency: input_.purchase.currency,per_unit_cost_basis: input_.purchase.per_unit_cost_basis,availability_policy: input_.purchase.availability_policy
    },tax_config: jsonCreditGrantTaxConfigToTransportTransform(input_.taxConfig),filters: {
      features: jsonArrayResourceKeyToTransportTransform(input_.filters.features)
    },priority: input_.priority
  }!;
}export function jsonCreateRequestNestedToApplicationTransform(
  input_?: any,
): CreateRequestNested {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),fundingMethod: input_.funding_method,currency: jsonCurrencyCodeToApplicationTransform(input_.currency),amount: input_.amount,purchase: {
      currency: input_.purchase.currency,perUnitCostBasis: input_.purchase.per_unit_cost_basis,availabilityPolicy: input_.purchase.availability_policy
    },taxConfig: jsonCreditGrantTaxConfigToApplicationTransform(input_.tax_config),filters: {
      features: jsonArrayResourceKeyToApplicationTransform(input_.filters.features)
    },priority: input_.priority
  }!;
}export function jsonCurrencyCodeToTransportTransform(
  input_?: CurrencyCode | null,
): any {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonCurrencyCodeToApplicationTransform(
  input_?: any,
): CurrencyCode {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonCreditGrantTaxConfigToTransportTransform(
  input_?: CreditGrantTaxConfig | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    behavior: input_.behavior,tax_code: jsonTaxCodeReferenceToTransportTransform(input_.taxCode)
  }!;
}export function jsonCreditGrantTaxConfigToApplicationTransform(
  input_?: any,
): CreditGrantTaxConfig {
  if(!input_) {
    return input_ as any;
  }
    return {
    behavior: input_.behavior,taxCode: jsonTaxCodeReferenceToApplicationTransform(input_.tax_code)
  }!;
}export function jsonTaxCodeReferenceToTransportTransform(
  input_?: TaxCodeReference | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonTaxCodeReferenceToApplicationTransform(
  input_?: any,
): TaxCodeReference {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonResourceReferenceToTransportTransform_2(
  input_?: ResourceReference_2 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonResourceReferenceToApplicationTransform_2(
  input_?: any,
): ResourceReference_2 {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonArrayResourceKeyToTransportTransform(
  items_?: Array<string> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayResourceKeyToApplicationTransform(
  items_?: any,
): Array<string> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonCreditGrantToTransportTransform(
  input_?: CreditGrant | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),funding_method: input_.fundingMethod,currency: jsonCurrencyCodeToTransportTransform(input_.currency),amount: input_.amount,purchase: {
      currency: input_.purchase.currency,per_unit_cost_basis: input_.purchase.per_unit_cost_basis,amount: input_.purchase.amount,availability_policy: input_.purchase.availability_policy,settlement_status: input_.purchase.settlement_status
    },tax_config: jsonCreditGrantTaxConfigToTransportTransform(input_.taxConfig),invoice: {
      id: input_.invoice.id,line: {
        id: input_.invoice.line.id
      }
    },filters: {
      features: jsonArrayResourceKeyToTransportTransform(input_.filters.features)
    },priority: input_.priority,voided_at: dateRfc3339Serializer(input_.voidedAt),status: input_.status
  }!;
}export function jsonCreditGrantToApplicationTransform(
  input_?: any,
): CreditGrant {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,fundingMethod: input_.funding_method,currency: jsonCurrencyCodeToApplicationTransform(input_.currency),amount: input_.amount,purchase: {
      currency: input_.purchase.currency,perUnitCostBasis: input_.purchase.per_unit_cost_basis,amount: input_.purchase.amount,availabilityPolicy: input_.purchase.availability_policy,settlementStatus: input_.purchase.settlement_status
    },taxConfig: jsonCreditGrantTaxConfigToApplicationTransform(input_.tax_config),invoice: {
      id: input_.invoice.id,line: {
        id: input_.invoice.line.id
      }
    },filters: {
      features: jsonArrayResourceKeyToApplicationTransform(input_.filters.features)
    },priority: input_.priority,voidedAt: dateDeserializer(input_.voided_at)!,status: input_.status
  }!;
}export function jsonListCreditGrantsParamsFilterToTransportTransform(
  input_?: ListCreditGrantsParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    status: input_.status,currency: input_.currency
  }!;
}export function jsonListCreditGrantsParamsFilterToApplicationTransform(
  input_?: any,
): ListCreditGrantsParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    status: input_.status,currency: input_.currency
  }!;
}export function jsonArrayCreditGrantToTransportTransform(
  items_?: Array<CreditGrant> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonCreditGrantToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayCreditGrantToApplicationTransform(
  items_?: any,
): Array<CreditGrant> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonCreditGrantToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonGetCreditBalanceParamsFilterToTransportTransform(
  input_?: GetCreditBalanceParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    currency: jsonCurrencyCodeToTransportTransform(input_.currency)
  }!;
}export function jsonGetCreditBalanceParamsFilterToApplicationTransform(
  input_?: any,
): GetCreditBalanceParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    currency: jsonCurrencyCodeToApplicationTransform(input_.currency)
  }!;
}export function jsonCreditBalancesToTransportTransform(
  input_?: CreditBalances | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    retrieved_at: dateRfc3339Serializer(input_.retrievedAt),balances: jsonArrayCreditBalanceToTransportTransform(input_.balances)
  }!;
}export function jsonCreditBalancesToApplicationTransform(
  input_?: any,
): CreditBalances {
  if(!input_) {
    return input_ as any;
  }
    return {
    retrievedAt: dateDeserializer(input_.retrieved_at)!,balances: jsonArrayCreditBalanceToApplicationTransform(input_.balances)
  }!;
}export function jsonArrayCreditBalanceToTransportTransform(
  items_?: Array<CreditBalance> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonCreditBalanceToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayCreditBalanceToApplicationTransform(
  items_?: any,
): Array<CreditBalance> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonCreditBalanceToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonCreditBalanceToTransportTransform(
  input_?: CreditBalance | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    currency: jsonCurrencyCodeToTransportTransform(input_.currency),pending: input_.pending,available: input_.available
  }!;
}export function jsonCreditBalanceToApplicationTransform(
  input_?: any,
): CreditBalance {
  if(!input_) {
    return input_ as any;
  }
    return {
    currency: jsonCurrencyCodeToApplicationTransform(input_.currency),pending: input_.pending,available: input_.available
  }!;
}export function jsonCreateRequestToTransportTransform_3(
  input_?: CreateRequest_9 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),currency: jsonCurrencyCodeToTransportTransform(input_.currency),amount: input_.amount
  }!;
}export function jsonCreateRequestToApplicationTransform_3(
  input_?: any,
): CreateRequest_9 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),currency: jsonCurrencyCodeToApplicationTransform(input_.currency),amount: input_.amount
  }!;
}export function jsonCreditAdjustmentToTransportTransform(
  input_?: CreditAdjustment | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),currency: jsonCurrencyCodeToTransportTransform(input_.currency),amount: input_.amount
  }!;
}export function jsonCreditAdjustmentToApplicationTransform(
  input_?: any,
): CreditAdjustment {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),currency: jsonCurrencyCodeToApplicationTransform(input_.currency),amount: input_.amount
  }!;
}export function jsonUpdateCreditGrantExternalSettlementRequestToTransportTransform(
  input_?: UpdateCreditGrantExternalSettlementRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    status: input_.status
  }!;
}export function jsonUpdateCreditGrantExternalSettlementRequestToApplicationTransform(
  input_?: any,
): UpdateCreditGrantExternalSettlementRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    status: input_.status
  }!;
}export function jsonListCreditTransactionsParamsFilterToTransportTransform(
  input_?: ListCreditTransactionsParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,currency: jsonCurrencyCodeToTransportTransform(input_.currency)
  }!;
}export function jsonListCreditTransactionsParamsFilterToApplicationTransform(
  input_?: any,
): ListCreditTransactionsParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,currency: jsonCurrencyCodeToApplicationTransform(input_.currency)
  }!;
}export function jsonArrayCreditTransactionToTransportTransform(
  items_?: Array<CreditTransaction> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonCreditTransactionToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayCreditTransactionToApplicationTransform(
  items_?: any,
): Array<CreditTransaction> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonCreditTransactionToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonCreditTransactionToTransportTransform(
  input_?: CreditTransaction | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),booked_at: dateRfc3339Serializer(input_.bookedAt),type: input_.type,currency: jsonCurrencyCodeToTransportTransform(input_.currency),amount: input_.amount,available_balance: {
      before: input_.availableBalance.before,after: input_.availableBalance.after
    }
  }!;
}export function jsonCreditTransactionToApplicationTransform(
  input_?: any,
): CreditTransaction {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,bookedAt: dateDeserializer(input_.booked_at)!,type: input_.type,currency: jsonCurrencyCodeToApplicationTransform(input_.currency),amount: input_.amount,availableBalance: {
      before: input_.available_balance.before,after: input_.available_balance.after
    }
  }!;
}export function jsonListCustomerChargesParamsFilterToTransportTransform(
  input_?: ListCustomerChargesParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    status: jsonStringFieldFilterExactToTransportTransform(input_.status)
  }!;
}export function jsonListCustomerChargesParamsFilterToApplicationTransform(
  input_?: any,
): ListCustomerChargesParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    status: jsonStringFieldFilterExactToApplicationTransform(input_.status)
  }!;
}export function jsonStringFieldFilterExactToTransportTransform(
  input_?: StringFieldFilterExact | null,
): any {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonStringFieldFilterExactToApplicationTransform(
  input_?: any,
): StringFieldFilterExact {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonArrayChargesExpandToTransportTransform(
  items_?: Array<ChargesExpand> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayChargesExpandToApplicationTransform(
  items_?: any,
): Array<ChargesExpand> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayChargeToTransportTransform(
  items_?: Array<Charge> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonChargeToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayChargeToApplicationTransform(
  items_?: any,
): Array<Charge> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonChargeToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonChargeToTransportDiscriminator(input_?: Charge): any {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.type;if( discriminatorValue === "flat_fee") {
    return jsonFlatFeeChargeToTransportTransform(input_ as any)!
  }

  if( discriminatorValue === "usage_based") {
    return jsonUsageBasedChargeToTransportTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonChargeToTransportTransform(input_?: Charge | null): any {
  if(!input_) {
    return input_ as any;
  }return jsonChargeToTransportDiscriminator(input_)
}export function jsonChargeToApplicationDiscriminator(input_?: any): Charge {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.type;if( discriminatorValue === "flat_fee") {
    return jsonFlatFeeChargeToApplicationTransform(input_ as any)!
  }

  if( discriminatorValue === "usage_based") {
    return jsonUsageBasedChargeToApplicationTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonChargeToApplicationTransform(input_?: any): Charge {
  if(!input_) {
    return input_ as any;
  }return jsonChargeToApplicationDiscriminator(input_)
}export function jsonFlatFeeChargeToTransportTransform(
  input_?: FlatFeeCharge | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),type: input_.type,customer: jsonCustomerReferenceToTransportTransform(input_.customer),managed_by: input_.managedBy,subscription: jsonSubscriptionReferenceToTransportTransform(input_.subscription),currency: input_.currency,status: input_.status,invoice_at: dateRfc3339Serializer(input_.invoiceAt),service_period: jsonClosedPeriodToTransportTransform(input_.servicePeriod),full_service_period: jsonClosedPeriodToTransportTransform(input_.fullServicePeriod),billing_period: jsonClosedPeriodToTransportTransform(input_.billingPeriod),advance_after: dateRfc3339Serializer(input_.advanceAfter),price: jsonPriceToTransportTransform(input_.price),unique_reference_id: input_.uniqueReferenceId,settlement_mode: input_.settlementMode,tax_config: jsonTaxConfigToTransportTransform(input_.taxConfig),payment_term: jsonPricePaymentTermToTransportTransform(input_.paymentTerm),discounts: jsonFlatFeeDiscountsToTransportTransform(input_.discounts),feature_key: input_.featureKey,proration_configuration: jsonProrationConfigurationToTransportTransform(input_.prorationConfiguration),amount_after_proration: jsonCurrencyAmountToTransportTransform(input_.amountAfterProration)
  }!;
}export function jsonFlatFeeChargeToApplicationTransform(
  input_?: any,
): FlatFeeCharge {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,type: input_.type,customer: jsonCustomerReferenceToApplicationTransform(input_.customer),managedBy: input_.managed_by,subscription: jsonSubscriptionReferenceToApplicationTransform(input_.subscription),currency: input_.currency,status: input_.status,invoiceAt: dateDeserializer(input_.invoice_at)!,servicePeriod: jsonClosedPeriodToApplicationTransform(input_.service_period),fullServicePeriod: jsonClosedPeriodToApplicationTransform(input_.full_service_period),billingPeriod: jsonClosedPeriodToApplicationTransform(input_.billing_period),advanceAfter: dateDeserializer(input_.advance_after)!,price: jsonPriceToApplicationTransform(input_.price),uniqueReferenceId: input_.unique_reference_id,settlementMode: input_.settlement_mode,taxConfig: jsonTaxConfigToApplicationTransform(input_.tax_config),paymentTerm: jsonPricePaymentTermToApplicationTransform(input_.payment_term),discounts: jsonFlatFeeDiscountsToApplicationTransform(input_.discounts),featureKey: input_.feature_key,prorationConfiguration: jsonProrationConfigurationToApplicationTransform(input_.proration_configuration),amountAfterProration: jsonCurrencyAmountToApplicationTransform(input_.amount_after_proration)
  }!;
}export function jsonCustomerReferenceToTransportTransform(
  input_?: CustomerReference | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonCustomerReferenceToApplicationTransform(
  input_?: any,
): CustomerReference {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonSubscriptionReferenceToTransportTransform(
  input_?: SubscriptionReference | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,phase: {
      id: input_.phase.id,item: {
        id: input_.phase.item.id
      }
    }
  }!;
}export function jsonSubscriptionReferenceToApplicationTransform(
  input_?: any,
): SubscriptionReference {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,phase: {
      id: input_.phase.id,item: {
        id: input_.phase.item.id
      }
    }
  }!;
}export function jsonClosedPeriodToTransportTransform(
  input_?: ClosedPeriod | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    from: dateRfc3339Serializer(input_.from),to: dateRfc3339Serializer(input_.to)
  }!;
}export function jsonClosedPeriodToApplicationTransform(
  input_?: any,
): ClosedPeriod {
  if(!input_) {
    return input_ as any;
  }
    return {
    from: dateDeserializer(input_.from)!,to: dateDeserializer(input_.to)!
  }!;
}export function jsonPriceToTransportDiscriminator(input_?: Price): any {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.type;if( discriminatorValue === "free") {
    return jsonPriceFreeToTransportTransform(input_ as any)!
  }

  if( discriminatorValue === "flat") {
    return jsonPriceFlatToTransportTransform(input_ as any)!
  }

  if( discriminatorValue === "unit") {
    return jsonPriceUnitToTransportTransform(input_ as any)!
  }

  if( discriminatorValue === "graduated") {
    return jsonPriceGraduatedToTransportTransform(input_ as any)!
  }

  if( discriminatorValue === "volume") {
    return jsonPriceVolumeToTransportTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonPriceToTransportTransform(input_?: Price | null): any {
  if(!input_) {
    return input_ as any;
  }return jsonPriceToTransportDiscriminator(input_)
}export function jsonPriceToApplicationDiscriminator(input_?: any): Price {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.type;if( discriminatorValue === "free") {
    return jsonPriceFreeToApplicationTransform(input_ as any)!
  }

  if( discriminatorValue === "flat") {
    return jsonPriceFlatToApplicationTransform(input_ as any)!
  }

  if( discriminatorValue === "unit") {
    return jsonPriceUnitToApplicationTransform(input_ as any)!
  }

  if( discriminatorValue === "graduated") {
    return jsonPriceGraduatedToApplicationTransform(input_ as any)!
  }

  if( discriminatorValue === "volume") {
    return jsonPriceVolumeToApplicationTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonPriceToApplicationTransform(input_?: any): Price {
  if(!input_) {
    return input_ as any;
  }return jsonPriceToApplicationDiscriminator(input_)
}export function jsonPriceFreeToTransportTransform(
  input_?: PriceFree | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type
  }!;
}export function jsonPriceFreeToApplicationTransform(input_?: any): PriceFree {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type
  }!;
}export function jsonPriceFlatToTransportTransform(
  input_?: PriceFlat | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,amount: input_.amount
  }!;
}export function jsonPriceFlatToApplicationTransform(input_?: any): PriceFlat {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,amount: input_.amount
  }!;
}export function jsonPriceUnitToTransportTransform(
  input_?: PriceUnit | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,amount: input_.amount
  }!;
}export function jsonPriceUnitToApplicationTransform(input_?: any): PriceUnit {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,amount: input_.amount
  }!;
}export function jsonPriceGraduatedToTransportTransform(
  input_?: PriceGraduated | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,tiers: jsonArrayPriceTierToTransportTransform(input_.tiers)
  }!;
}export function jsonPriceGraduatedToApplicationTransform(
  input_?: any,
): PriceGraduated {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,tiers: jsonArrayPriceTierToApplicationTransform(input_.tiers)
  }!;
}export function jsonArrayPriceTierToTransportTransform(
  items_?: Array<PriceTier> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonPriceTierToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayPriceTierToApplicationTransform(
  items_?: any,
): Array<PriceTier> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonPriceTierToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonPriceTierToTransportTransform(
  input_?: PriceTier | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    up_to_amount: input_.upToAmount,flat_price: jsonPriceFlatToTransportTransform(input_.flatPrice),unit_price: jsonPriceUnitToTransportTransform(input_.unitPrice)
  }!;
}export function jsonPriceTierToApplicationTransform(input_?: any): PriceTier {
  if(!input_) {
    return input_ as any;
  }
    return {
    upToAmount: input_.up_to_amount,flatPrice: jsonPriceFlatToApplicationTransform(input_.flat_price),unitPrice: jsonPriceUnitToApplicationTransform(input_.unit_price)
  }!;
}export function jsonPriceVolumeToTransportTransform(
  input_?: PriceVolume | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,tiers: jsonArrayPriceTierToTransportTransform(input_.tiers)
  }!;
}export function jsonPriceVolumeToApplicationTransform(
  input_?: any,
): PriceVolume {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,tiers: jsonArrayPriceTierToApplicationTransform(input_.tiers)
  }!;
}export function jsonTaxConfigToTransportTransform(
  input_?: TaxConfig | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    behavior: input_.behavior,stripe: jsonTaxConfigStripeToTransportTransform(input_.stripe),external_invoicing: jsonTaxConfigExternalInvoicingToTransportTransform(input_.externalInvoicing),tax_code_id: input_.taxCodeId,tax_code: jsonTaxCodeReferenceToTransportTransform(input_.taxCode)
  }!;
}export function jsonTaxConfigToApplicationTransform(input_?: any): TaxConfig {
  if(!input_) {
    return input_ as any;
  }
    return {
    behavior: input_.behavior,stripe: jsonTaxConfigStripeToApplicationTransform(input_.stripe),externalInvoicing: jsonTaxConfigExternalInvoicingToApplicationTransform(input_.external_invoicing),taxCodeId: input_.tax_code_id,taxCode: jsonTaxCodeReferenceToApplicationTransform(input_.tax_code)
  }!;
}export function jsonTaxConfigStripeToTransportTransform(
  input_?: TaxConfigStripe | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code
  }!;
}export function jsonTaxConfigStripeToApplicationTransform(
  input_?: any,
): TaxConfigStripe {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code
  }!;
}export function jsonTaxConfigExternalInvoicingToTransportTransform(
  input_?: TaxConfigExternalInvoicing | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code
  }!;
}export function jsonTaxConfigExternalInvoicingToApplicationTransform(
  input_?: any,
): TaxConfigExternalInvoicing {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code
  }!;
}export function jsonPricePaymentTermToTransportTransform(
  input_?: PricePaymentTerm | null,
): any {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonPricePaymentTermToApplicationTransform(
  input_?: any,
): PricePaymentTerm {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonFlatFeeDiscountsToTransportTransform(
  input_?: FlatFeeDiscounts | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    percentage: input_.percentage
  }!;
}export function jsonFlatFeeDiscountsToApplicationTransform(
  input_?: any,
): FlatFeeDiscounts {
  if(!input_) {
    return input_ as any;
  }
    return {
    percentage: input_.percentage
  }!;
}export function jsonProrationConfigurationToTransportTransform(
  input_?: ProrationConfiguration | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    mode: input_.mode
  }!;
}export function jsonProrationConfigurationToApplicationTransform(
  input_?: any,
): ProrationConfiguration {
  if(!input_) {
    return input_ as any;
  }
    return {
    mode: input_.mode
  }!;
}export function jsonCurrencyAmountToTransportTransform(
  input_?: CurrencyAmount | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    amount: input_.amount,currency: input_.currency
  }!;
}export function jsonCurrencyAmountToApplicationTransform(
  input_?: any,
): CurrencyAmount {
  if(!input_) {
    return input_ as any;
  }
    return {
    amount: input_.amount,currency: input_.currency
  }!;
}export function jsonUsageBasedChargeToTransportTransform(
  input_?: UsageBasedCharge | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),type: input_.type,customer: jsonCustomerReferenceToTransportTransform(input_.customer),managed_by: input_.managedBy,subscription: jsonSubscriptionReferenceToTransportTransform(input_.subscription),currency: input_.currency,status: input_.status,invoice_at: dateRfc3339Serializer(input_.invoiceAt),service_period: jsonClosedPeriodToTransportTransform(input_.servicePeriod),full_service_period: jsonClosedPeriodToTransportTransform(input_.fullServicePeriod),billing_period: jsonClosedPeriodToTransportTransform(input_.billingPeriod),advance_after: dateRfc3339Serializer(input_.advanceAfter),price: jsonPriceToTransportTransform(input_.price),unique_reference_id: input_.uniqueReferenceId,settlement_mode: input_.settlementMode,tax_config: jsonTaxConfigToTransportTransform(input_.taxConfig),discounts: jsonDiscountsToTransportTransform(input_.discounts),feature_key: input_.featureKey,totals: jsonChargeTotalsToTransportTransform(input_.totals)
  }!;
}export function jsonUsageBasedChargeToApplicationTransform(
  input_?: any,
): UsageBasedCharge {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,type: input_.type,customer: jsonCustomerReferenceToApplicationTransform(input_.customer),managedBy: input_.managed_by,subscription: jsonSubscriptionReferenceToApplicationTransform(input_.subscription),currency: input_.currency,status: input_.status,invoiceAt: dateDeserializer(input_.invoice_at)!,servicePeriod: jsonClosedPeriodToApplicationTransform(input_.service_period),fullServicePeriod: jsonClosedPeriodToApplicationTransform(input_.full_service_period),billingPeriod: jsonClosedPeriodToApplicationTransform(input_.billing_period),advanceAfter: dateDeserializer(input_.advance_after)!,price: jsonPriceToApplicationTransform(input_.price),uniqueReferenceId: input_.unique_reference_id,settlementMode: input_.settlement_mode,taxConfig: jsonTaxConfigToApplicationTransform(input_.tax_config),discounts: jsonDiscountsToApplicationTransform(input_.discounts),featureKey: input_.feature_key,totals: jsonChargeTotalsToApplicationTransform(input_.totals)
  }!;
}export function jsonDiscountsToTransportTransform(
  input_?: Discounts | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    percentage: input_.percentage,usage: input_.usage
  }!;
}export function jsonDiscountsToApplicationTransform(input_?: any): Discounts {
  if(!input_) {
    return input_ as any;
  }
    return {
    percentage: input_.percentage,usage: input_.usage
  }!;
}export function jsonChargeTotalsToTransportTransform(
  input_?: ChargeTotals | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    booked: jsonBillingTotalsToTransportTransform(input_.booked),realtime: jsonBillingTotalsToTransportTransform(input_.realtime)
  }!;
}export function jsonChargeTotalsToApplicationTransform(
  input_?: any,
): ChargeTotals {
  if(!input_) {
    return input_ as any;
  }
    return {
    booked: jsonBillingTotalsToApplicationTransform(input_.booked),realtime: jsonBillingTotalsToApplicationTransform(input_.realtime)
  }!;
}export function jsonBillingTotalsToTransportTransform(
  input_?: BillingTotals | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    amount: input_.amount,taxes_total: input_.taxesTotal,taxes_inclusive_total: input_.taxesInclusiveTotal,taxes_exclusive_total: input_.taxesExclusiveTotal,charges_total: input_.chargesTotal,discounts_total: input_.discountsTotal,credits_total: input_.creditsTotal,total: input_.total
  }!;
}export function jsonBillingTotalsToApplicationTransform(
  input_?: any,
): BillingTotals {
  if(!input_) {
    return input_ as any;
  }
    return {
    amount: input_.amount,taxesTotal: input_.taxes_total,taxesInclusiveTotal: input_.taxes_inclusive_total,taxesExclusiveTotal: input_.taxes_exclusive_total,chargesTotal: input_.charges_total,discountsTotal: input_.discounts_total,creditsTotal: input_.credits_total,total: input_.total
  }!;
}export function jsonSubscriptionCreateToTransportTransform(
  input_?: SubscriptionCreate | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    labels: jsonLabelsToTransportTransform(input_.labels),customer: {
      id: input_.customer.id,key: input_.customer.key
    },plan: {
      id: input_.plan.id,key: input_.plan.key,version: input_.plan.version
    },billing_anchor: dateRfc3339Serializer(input_.billingAnchor)
  }!;
}export function jsonSubscriptionCreateToApplicationTransform(
  input_?: any,
): SubscriptionCreate {
  if(!input_) {
    return input_ as any;
  }
    return {
    labels: jsonLabelsToApplicationTransform(input_.labels),customer: {
      id: input_.customer.id,key: input_.customer.key
    },plan: {
      id: input_.plan.id,key: input_.plan.key,version: input_.plan.version
    },billingAnchor: dateDeserializer(input_.billing_anchor)!
  }!;
}export function jsonSubscriptionToTransportTransform(
  input_?: Subscription | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),customer_id: input_.customerId,plan_id: input_.planId,billing_anchor: dateRfc3339Serializer(input_.billingAnchor),status: input_.status
  }!;
}export function jsonSubscriptionToApplicationTransform(
  input_?: any,
): Subscription {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,customerId: input_.customer_id,planId: input_.plan_id,billingAnchor: dateDeserializer(input_.billing_anchor)!,status: input_.status
  }!;
}export function jsonListSubscriptionsParamsFilterToTransportTransform(
  input_?: ListSubscriptionsParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: jsonUlidFieldFilterToTransportTransform(input_.id),customer_id: jsonUlidFieldFilterToTransportTransform(input_.customerId),status: jsonStringFieldFilterExactToTransportTransform(input_.status),plan_id: jsonUlidFieldFilterToTransportTransform(input_.planId),plan_key: jsonStringFieldFilterExactToTransportTransform(input_.planKey)
  }!;
}export function jsonListSubscriptionsParamsFilterToApplicationTransform(
  input_?: any,
): ListSubscriptionsParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: jsonUlidFieldFilterToApplicationTransform(input_.id),customerId: jsonUlidFieldFilterToApplicationTransform(input_.customer_id),status: jsonStringFieldFilterExactToApplicationTransform(input_.status),planId: jsonUlidFieldFilterToApplicationTransform(input_.plan_id),planKey: jsonStringFieldFilterExactToApplicationTransform(input_.plan_key)
  }!;
}export function jsonArraySubscriptionToTransportTransform(
  items_?: Array<Subscription> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonSubscriptionToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArraySubscriptionToApplicationTransform(
  items_?: any,
): Array<Subscription> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonSubscriptionToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonSubscriptionCancelToTransportTransform(
  input_?: SubscriptionCancel | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    timing: jsonSubscriptionEditTimingToTransportTransform(input_.timing)
  }!;
}export function jsonSubscriptionCancelToApplicationTransform(
  input_?: any,
): SubscriptionCancel {
  if(!input_) {
    return input_ as any;
  }
    return {
    timing: jsonSubscriptionEditTimingToApplicationTransform(input_.timing)
  }!;
}export function jsonSubscriptionEditTimingToTransportTransform(
  input_?: SubscriptionEditTiming | null,
): any {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonSubscriptionEditTimingToApplicationTransform(
  input_?: any,
): SubscriptionEditTiming {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonSubscriptionChangeToTransportTransform(
  input_?: SubscriptionChange | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    labels: jsonLabelsToTransportTransform(input_.labels),customer: {
      id: input_.customer.id,key: input_.customer.key
    },plan: {
      id: input_.plan.id,key: input_.plan.key,version: input_.plan.version
    },billing_anchor: dateRfc3339Serializer(input_.billingAnchor),timing: jsonSubscriptionEditTimingToTransportTransform(input_.timing)
  }!;
}export function jsonSubscriptionChangeToApplicationTransform(
  input_?: any,
): SubscriptionChange {
  if(!input_) {
    return input_ as any;
  }
    return {
    labels: jsonLabelsToApplicationTransform(input_.labels),customer: {
      id: input_.customer.id,key: input_.customer.key
    },plan: {
      id: input_.plan.id,key: input_.plan.key,version: input_.plan.version
    },billingAnchor: dateDeserializer(input_.billing_anchor)!,timing: jsonSubscriptionEditTimingToApplicationTransform(input_.timing)
  }!;
}export function jsonSubscriptionChangeResponseToTransportTransform(
  input_?: SubscriptionChangeResponse | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    current: jsonSubscriptionToTransportTransform(input_.current),next: jsonSubscriptionToTransportTransform(input_.next)
  }!;
}export function jsonSubscriptionChangeResponseToApplicationTransform(
  input_?: any,
): SubscriptionChangeResponse {
  if(!input_) {
    return input_ as any;
  }
    return {
    current: jsonSubscriptionToApplicationTransform(input_.current),next: jsonSubscriptionToApplicationTransform(input_.next)
  }!;
}export function jsonArraySubscriptionAddonToTransportTransform(
  items_?: Array<SubscriptionAddon> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonSubscriptionAddonToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArraySubscriptionAddonToApplicationTransform(
  items_?: any,
): Array<SubscriptionAddon> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonSubscriptionAddonToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonSubscriptionAddonToTransportTransform(
  input_?: SubscriptionAddon | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),addon: jsonResourceReferenceToTransportTransform_3(input_.addon),quantity: input_.quantity,quantity_at: dateRfc3339Serializer(input_.quantityAt),active_from: dateRfc3339Serializer(input_.activeFrom),active_to: dateRfc3339Serializer(input_.activeTo)
  }!;
}export function jsonSubscriptionAddonToApplicationTransform(
  input_?: any,
): SubscriptionAddon {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,addon: jsonResourceReferenceToApplicationTransform_3(input_.addon),quantity: input_.quantity,quantityAt: dateDeserializer(input_.quantity_at)!,activeFrom: dateDeserializer(input_.active_from)!,activeTo: dateDeserializer(input_.active_to)!
  }!;
}export function jsonResourceReferenceToTransportTransform_3(
  input_?: ResourceReference_3 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonResourceReferenceToApplicationTransform_3(
  input_?: any,
): ResourceReference_3 {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonArrayAppToTransportTransform(
  items_?: Array<App> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonAppToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayAppToApplicationTransform(items_?: any): Array<App> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonAppToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonAppToTransportDiscriminator(input_?: App): any {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.type;if( discriminatorValue === "stripe") {
    return jsonAppStripeToTransportTransform(input_ as any)!
  }

  if( discriminatorValue === "sandbox") {
    return jsonAppSandboxToTransportTransform(input_ as any)!
  }

  if( discriminatorValue === "external_invoicing") {
    return jsonAppExternalInvoicingToTransportTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonAppToTransportTransform(input_?: App | null): any {
  if(!input_) {
    return input_ as any;
  }return jsonAppToTransportDiscriminator(input_)
}export function jsonAppToApplicationDiscriminator(input_?: any): App {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.type;if( discriminatorValue === "stripe") {
    return jsonAppStripeToApplicationTransform(input_ as any)!
  }

  if( discriminatorValue === "sandbox") {
    return jsonAppSandboxToApplicationTransform(input_ as any)!
  }

  if( discriminatorValue === "external_invoicing") {
    return jsonAppExternalInvoicingToApplicationTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonAppToApplicationTransform(input_?: any): App {
  if(!input_) {
    return input_ as any;
  }return jsonAppToApplicationDiscriminator(input_)
}export function jsonAppStripeToTransportTransform(
  input_?: AppStripe | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),type: input_.type,definition: jsonAppCatalogItemToTransportTransform(input_.definition),status: input_.status,account_id: input_.accountId,livemode: input_.livemode,masked_api_key: input_.maskedApiKey,secret_api_key: input_.secretApiKey
  }!;
}export function jsonAppStripeToApplicationTransform(input_?: any): AppStripe {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,type: input_.type,definition: jsonAppCatalogItemToApplicationTransform(input_.definition),status: input_.status,accountId: input_.account_id,livemode: input_.livemode,maskedApiKey: input_.masked_api_key,secretApiKey: input_.secret_api_key
  }!;
}export function jsonAppCatalogItemToTransportTransform(
  input_?: AppCatalogItem | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,name: input_.name,description: input_.description
  }!;
}export function jsonAppCatalogItemToApplicationTransform(
  input_?: any,
): AppCatalogItem {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,name: input_.name,description: input_.description
  }!;
}export function jsonAppSandboxToTransportTransform(
  input_?: AppSandbox | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),type: input_.type,definition: jsonAppCatalogItemToTransportTransform(input_.definition),status: input_.status
  }!;
}export function jsonAppSandboxToApplicationTransform(
  input_?: any,
): AppSandbox {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,type: input_.type,definition: jsonAppCatalogItemToApplicationTransform(input_.definition),status: input_.status
  }!;
}export function jsonAppExternalInvoicingToTransportTransform(
  input_?: AppExternalInvoicing | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),type: input_.type,definition: jsonAppCatalogItemToTransportTransform(input_.definition),status: input_.status,enable_draft_sync_hook: input_.enableDraftSyncHook,enable_issuing_sync_hook: input_.enableIssuingSyncHook
  }!;
}export function jsonAppExternalInvoicingToApplicationTransform(
  input_?: any,
): AppExternalInvoicing {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,type: input_.type,definition: jsonAppCatalogItemToApplicationTransform(input_.definition),status: input_.status,enableDraftSyncHook: input_.enable_draft_sync_hook,enableIssuingSyncHook: input_.enable_issuing_sync_hook
  }!;
}export function jsonArrayBillingProfileToTransportTransform(
  items_?: Array<BillingProfile> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonBillingProfileToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayBillingProfileToApplicationTransform(
  items_?: any,
): Array<BillingProfile> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonBillingProfileToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonBillingProfileToTransportTransform(
  input_?: BillingProfile | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),supplier: jsonBillingPartyToTransportTransform(input_.supplier),workflow: jsonBillingWorkflowToTransportTransform(input_.workflow),apps: jsonBillingProfileAppReferencesToTransportTransform(input_.apps),default: input_.default_
  }!;
}export function jsonBillingProfileToApplicationTransform(
  input_?: any,
): BillingProfile {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,supplier: jsonBillingPartyToApplicationTransform(input_.supplier),workflow: jsonBillingWorkflowToApplicationTransform(input_.workflow),apps: jsonBillingProfileAppReferencesToApplicationTransform(input_.apps),default_: input_.default
  }!;
}export function jsonBillingPartyToTransportTransform(
  input_?: BillingParty | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,key: input_.key,name: input_.name,tax_id: jsonBillingPartyTaxIdentityToTransportTransform(input_.taxId),addresses: jsonBillingPartyAddressesToTransportTransform(input_.addresses)
  }!;
}export function jsonBillingPartyToApplicationTransform(
  input_?: any,
): BillingParty {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,key: input_.key,name: input_.name,taxId: jsonBillingPartyTaxIdentityToApplicationTransform(input_.tax_id),addresses: jsonBillingPartyAddressesToApplicationTransform(input_.addresses)
  }!;
}export function jsonBillingPartyTaxIdentityToTransportTransform(
  input_?: BillingPartyTaxIdentity | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code
  }!;
}export function jsonBillingPartyTaxIdentityToApplicationTransform(
  input_?: any,
): BillingPartyTaxIdentity {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code
  }!;
}export function jsonBillingPartyAddressesToTransportTransform(
  input_?: BillingPartyAddresses | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    billing_address: jsonAddressToTransportTransform_2(input_.billingAddress)
  }!;
}export function jsonBillingPartyAddressesToApplicationTransform(
  input_?: any,
): BillingPartyAddresses {
  if(!input_) {
    return input_ as any;
  }
    return {
    billingAddress: jsonAddressToApplicationTransform_2(input_.billing_address)
  }!;
}export function jsonAddressToTransportTransform_2(
  input_?: Address_2 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    country: input_.country,postal_code: input_.postalCode,state: input_.state,city: input_.city,line1: input_.line1,line2: input_.line2,phone_number: input_.phoneNumber
  }!;
}export function jsonAddressToApplicationTransform_2(input_?: any): Address_2 {
  if(!input_) {
    return input_ as any;
  }
    return {
    country: input_.country,postalCode: input_.postal_code,state: input_.state,city: input_.city,line1: input_.line1,line2: input_.line2,phoneNumber: input_.phone_number
  }!;
}export function jsonBillingWorkflowToTransportTransform(
  input_?: BillingWorkflow | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    collection: jsonBillingWorkflowCollectionSettingsToTransportTransform(input_.collection),invoicing: jsonBillingWorkflowInvoicingSettingsToTransportTransform(input_.invoicing),payment: jsonBillingWorkflowPaymentSettingsToTransportTransform(input_.payment),tax: jsonBillingWorkflowTaxSettingsToTransportTransform(input_.tax)
  }!;
}export function jsonBillingWorkflowToApplicationTransform(
  input_?: any,
): BillingWorkflow {
  if(!input_) {
    return input_ as any;
  }
    return {
    collection: jsonBillingWorkflowCollectionSettingsToApplicationTransform(input_.collection),invoicing: jsonBillingWorkflowInvoicingSettingsToApplicationTransform(input_.invoicing),payment: jsonBillingWorkflowPaymentSettingsToApplicationTransform(input_.payment),tax: jsonBillingWorkflowTaxSettingsToApplicationTransform(input_.tax)
  }!;
}export function jsonBillingWorkflowCollectionSettingsToTransportTransform(
  input_?: BillingWorkflowCollectionSettings | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    alignment: jsonBillingWorkflowCollectionAlignmentToTransportTransform(input_.alignment),interval: input_.interval
  }!;
}export function jsonBillingWorkflowCollectionSettingsToApplicationTransform(
  input_?: any,
): BillingWorkflowCollectionSettings {
  if(!input_) {
    return input_ as any;
  }
    return {
    alignment: jsonBillingWorkflowCollectionAlignmentToApplicationTransform(input_.alignment),interval: input_.interval
  }!;
}export function jsonBillingWorkflowCollectionAlignmentToTransportDiscriminator(
  input_?: BillingWorkflowCollectionAlignment,
): any {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.type;if( discriminatorValue === "subscription") {
    return jsonBillingWorkflowCollectionAlignmentSubscriptionToTransportTransform(input_ as any)!
  }

  if( discriminatorValue === "anchored") {
    return jsonBillingWorkflowCollectionAlignmentAnchoredToTransportTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonBillingWorkflowCollectionAlignmentToTransportTransform(
  input_?: BillingWorkflowCollectionAlignment | null,
): any {
  if(!input_) {
    return input_ as any;
  }return jsonBillingWorkflowCollectionAlignmentToTransportDiscriminator(input_)
}export function jsonBillingWorkflowCollectionAlignmentToApplicationDiscriminator(
  input_?: any,
): BillingWorkflowCollectionAlignment {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.type;if( discriminatorValue === "subscription") {
    return jsonBillingWorkflowCollectionAlignmentSubscriptionToApplicationTransform(input_ as any)!
  }

  if( discriminatorValue === "anchored") {
    return jsonBillingWorkflowCollectionAlignmentAnchoredToApplicationTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonBillingWorkflowCollectionAlignmentToApplicationTransform(
  input_?: any,
): BillingWorkflowCollectionAlignment {
  if(!input_) {
    return input_ as any;
  }return jsonBillingWorkflowCollectionAlignmentToApplicationDiscriminator(input_)
}export function jsonBillingWorkflowCollectionAlignmentSubscriptionToTransportTransform(
  input_?: BillingWorkflowCollectionAlignmentSubscription | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type
  }!;
}export function jsonBillingWorkflowCollectionAlignmentSubscriptionToApplicationTransform(
  input_?: any,
): BillingWorkflowCollectionAlignmentSubscription {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type
  }!;
}export function jsonBillingWorkflowCollectionAlignmentAnchoredToTransportTransform(
  input_?: BillingWorkflowCollectionAlignmentAnchored | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,recurring_period: jsonRecurringPeriodToTransportTransform(input_.recurringPeriod)
  }!;
}export function jsonBillingWorkflowCollectionAlignmentAnchoredToApplicationTransform(
  input_?: any,
): BillingWorkflowCollectionAlignmentAnchored {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,recurringPeriod: jsonRecurringPeriodToApplicationTransform(input_.recurring_period)
  }!;
}export function jsonRecurringPeriodToTransportTransform(
  input_?: RecurringPeriod | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    anchor: dateRfc3339Serializer(input_.anchor),interval: input_.interval
  }!;
}export function jsonRecurringPeriodToApplicationTransform(
  input_?: any,
): RecurringPeriod {
  if(!input_) {
    return input_ as any;
  }
    return {
    anchor: dateDeserializer(input_.anchor)!,interval: input_.interval
  }!;
}export function jsonBillingWorkflowInvoicingSettingsToTransportTransform(
  input_?: BillingWorkflowInvoicingSettings | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    auto_advance: input_.autoAdvance,draft_period: input_.draftPeriod,progressive_billing: input_.progressiveBilling
  }!;
}export function jsonBillingWorkflowInvoicingSettingsToApplicationTransform(
  input_?: any,
): BillingWorkflowInvoicingSettings {
  if(!input_) {
    return input_ as any;
  }
    return {
    autoAdvance: input_.auto_advance,draftPeriod: input_.draft_period,progressiveBilling: input_.progressive_billing
  }!;
}export function jsonBillingWorkflowPaymentSettingsToTransportDiscriminator(
  input_?: BillingWorkflowPaymentSettings,
): any {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.collection_method;if( discriminatorValue === "charge_automatically") {
    return jsonBillingWorkflowPaymentChargeAutomaticallySettingsToTransportTransform(input_ as any)!
  }

  if( discriminatorValue === "send_invoice") {
    return jsonBillingWorkflowPaymentSendInvoiceSettingsToTransportTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonBillingWorkflowPaymentSettingsToTransportTransform(
  input_?: BillingWorkflowPaymentSettings | null,
): any {
  if(!input_) {
    return input_ as any;
  }return jsonBillingWorkflowPaymentSettingsToTransportDiscriminator(input_)
}export function jsonBillingWorkflowPaymentSettingsToApplicationDiscriminator(
  input_?: any,
): BillingWorkflowPaymentSettings {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.collection_method;if( discriminatorValue === "charge_automatically") {
    return jsonBillingWorkflowPaymentChargeAutomaticallySettingsToApplicationTransform(input_ as any)!
  }

  if( discriminatorValue === "send_invoice") {
    return jsonBillingWorkflowPaymentSendInvoiceSettingsToApplicationTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonBillingWorkflowPaymentSettingsToApplicationTransform(
  input_?: any,
): BillingWorkflowPaymentSettings {
  if(!input_) {
    return input_ as any;
  }return jsonBillingWorkflowPaymentSettingsToApplicationDiscriminator(input_)
}export function jsonBillingWorkflowPaymentChargeAutomaticallySettingsToTransportTransform(
  input_?: BillingWorkflowPaymentChargeAutomaticallySettings | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    collection_method: input_.collectionMethod
  }!;
}export function jsonBillingWorkflowPaymentChargeAutomaticallySettingsToApplicationTransform(
  input_?: any,
): BillingWorkflowPaymentChargeAutomaticallySettings {
  if(!input_) {
    return input_ as any;
  }
    return {
    collectionMethod: input_.collection_method
  }!;
}export function jsonBillingWorkflowPaymentSendInvoiceSettingsToTransportTransform(
  input_?: BillingWorkflowPaymentSendInvoiceSettings | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    collection_method: input_.collectionMethod,due_after: input_.dueAfter
  }!;
}export function jsonBillingWorkflowPaymentSendInvoiceSettingsToApplicationTransform(
  input_?: any,
): BillingWorkflowPaymentSendInvoiceSettings {
  if(!input_) {
    return input_ as any;
  }
    return {
    collectionMethod: input_.collection_method,dueAfter: input_.due_after
  }!;
}export function jsonBillingWorkflowTaxSettingsToTransportTransform(
  input_?: BillingWorkflowTaxSettings | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    enabled: input_.enabled,enforced: input_.enforced,default_tax_config: jsonTaxConfigToTransportTransform(input_.defaultTaxConfig)
  }!;
}export function jsonBillingWorkflowTaxSettingsToApplicationTransform(
  input_?: any,
): BillingWorkflowTaxSettings {
  if(!input_) {
    return input_ as any;
  }
    return {
    enabled: input_.enabled,enforced: input_.enforced,defaultTaxConfig: jsonTaxConfigToApplicationTransform(input_.default_tax_config)
  }!;
}export function jsonBillingProfileAppReferencesToTransportTransform(
  input_?: BillingProfileAppReferences | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    tax: jsonAppReferenceToTransportTransform(input_.tax),invoicing: jsonAppReferenceToTransportTransform(input_.invoicing),payment: jsonAppReferenceToTransportTransform(input_.payment)
  }!;
}export function jsonBillingProfileAppReferencesToApplicationTransform(
  input_?: any,
): BillingProfileAppReferences {
  if(!input_) {
    return input_ as any;
  }
    return {
    tax: jsonAppReferenceToApplicationTransform(input_.tax),invoicing: jsonAppReferenceToApplicationTransform(input_.invoicing),payment: jsonAppReferenceToApplicationTransform(input_.payment)
  }!;
}export function jsonAppReferenceToTransportTransform(
  input_?: AppReference | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonAppReferenceToApplicationTransform(
  input_?: any,
): AppReference {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonCreateRequestToTransportTransform_4(
  input_?: CreateRequest_8 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),supplier: jsonBillingPartyToTransportTransform(input_.supplier),workflow: jsonBillingWorkflowToTransportTransform(input_.workflow),apps: jsonBillingProfileAppReferencesToTransportTransform(input_.apps),default: input_.default_
  }!;
}export function jsonCreateRequestToApplicationTransform_4(
  input_?: any,
): CreateRequest_8 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),supplier: jsonBillingPartyToApplicationTransform(input_.supplier),workflow: jsonBillingWorkflowToApplicationTransform(input_.workflow),apps: jsonBillingProfileAppReferencesToApplicationTransform(input_.apps),default_: input_.default
  }!;
}export function jsonUpsertRequestToTransportTransform_4(
  input_?: UpsertRequest_5 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),supplier: jsonBillingPartyToTransportTransform(input_.supplier),workflow: jsonBillingWorkflowToTransportTransform(input_.workflow),default: input_.default_
  }!;
}export function jsonUpsertRequestToApplicationTransform_4(
  input_?: any,
): UpsertRequest_5 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),supplier: jsonBillingPartyToApplicationTransform(input_.supplier),workflow: jsonBillingWorkflowToApplicationTransform(input_.workflow),default_: input_.default
  }!;
}export function jsonCreateRequestToTransportTransform_5(
  input_?: CreateRequest_7 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),key: input_.key,app_mappings: jsonArrayTaxCodeAppMappingToTransportTransform(input_.appMappings)
  }!;
}export function jsonCreateRequestToApplicationTransform_5(
  input_?: any,
): CreateRequest_7 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),key: input_.key,appMappings: jsonArrayTaxCodeAppMappingToApplicationTransform(input_.app_mappings)
  }!;
}export function jsonArrayTaxCodeAppMappingToTransportTransform(
  items_?: Array<TaxCodeAppMapping> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonTaxCodeAppMappingToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayTaxCodeAppMappingToApplicationTransform(
  items_?: any,
): Array<TaxCodeAppMapping> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonTaxCodeAppMappingToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonTaxCodeAppMappingToTransportTransform(
  input_?: TaxCodeAppMapping | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    app_type: input_.appType,tax_code: input_.taxCode
  }!;
}export function jsonTaxCodeAppMappingToApplicationTransform(
  input_?: any,
): TaxCodeAppMapping {
  if(!input_) {
    return input_ as any;
  }
    return {
    appType: input_.app_type,taxCode: input_.tax_code
  }!;
}export function jsonTaxCodeToTransportTransform(input_?: TaxCode | null): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),key: input_.key,app_mappings: jsonArrayTaxCodeAppMappingToTransportTransform(input_.appMappings)
  }!;
}export function jsonTaxCodeToApplicationTransform(input_?: any): TaxCode {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,key: input_.key,appMappings: jsonArrayTaxCodeAppMappingToApplicationTransform(input_.app_mappings)
  }!;
}export function jsonArrayTaxCodeToTransportTransform(
  items_?: Array<TaxCode> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonTaxCodeToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayTaxCodeToApplicationTransform(
  items_?: any,
): Array<TaxCode> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonTaxCodeToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonUpsertRequestToTransportTransform_5(
  input_?: UpsertRequest_4 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),app_mappings: jsonArrayTaxCodeAppMappingToTransportTransform(input_.appMappings)
  }!;
}export function jsonUpsertRequestToApplicationTransform_5(
  input_?: any,
): UpsertRequest_4 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),appMappings: jsonArrayTaxCodeAppMappingToApplicationTransform(input_.app_mappings)
  }!;
}export function jsonListCurrenciesParamsFilterToTransportTransform(
  input_?: ListCurrenciesParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type
  }!;
}export function jsonListCurrenciesParamsFilterToApplicationTransform(
  input_?: any,
): ListCurrenciesParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type
  }!;
}export function jsonArrayCurrencyToTransportTransform(
  items_?: Array<Currency> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonCurrencyToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayCurrencyToApplicationTransform(
  items_?: any,
): Array<Currency> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonCurrencyToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonCurrencyToTransportDiscriminator(input_?: Currency): any {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.type;if( discriminatorValue === "fiat") {
    return jsonCurrencyFiatToTransportTransform(input_ as any)!
  }

  if( discriminatorValue === "custom") {
    return jsonCurrencyCustomToTransportTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonCurrencyToTransportTransform(
  input_?: Currency | null,
): any {
  if(!input_) {
    return input_ as any;
  }return jsonCurrencyToTransportDiscriminator(input_)
}export function jsonCurrencyToApplicationDiscriminator(
  input_?: any,
): Currency {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.type;if( discriminatorValue === "fiat") {
    return jsonCurrencyFiatToApplicationTransform(input_ as any)!
  }

  if( discriminatorValue === "custom") {
    return jsonCurrencyCustomToApplicationTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonCurrencyToApplicationTransform(input_?: any): Currency {
  if(!input_) {
    return input_ as any;
  }return jsonCurrencyToApplicationDiscriminator(input_)
}export function jsonCurrencyFiatToTransportTransform(
  input_?: CurrencyFiat | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,type: input_.type,name: input_.name,description: input_.description,symbol: input_.symbol,code: input_.code
  }!;
}export function jsonCurrencyFiatToApplicationTransform(
  input_?: any,
): CurrencyFiat {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,type: input_.type,name: input_.name,description: input_.description,symbol: input_.symbol,code: input_.code
  }!;
}export function jsonCurrencyCustomToTransportTransform(
  input_?: CurrencyCustom | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,type: input_.type,name: input_.name,description: input_.description,symbol: input_.symbol,code: input_.code,created_at: dateRfc3339Serializer(input_.createdAt)
  }!;
}export function jsonCurrencyCustomToApplicationTransform(
  input_?: any,
): CurrencyCustom {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,type: input_.type,name: input_.name,description: input_.description,symbol: input_.symbol,code: input_.code,createdAt: dateDeserializer(input_.created_at)!
  }!;
}export function jsonCreateRequestToTransportTransform_6(
  input_?: CreateRequest_6 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,symbol: input_.symbol,code: input_.code
  }!;
}export function jsonCreateRequestToApplicationTransform_6(
  input_?: any,
): CreateRequest_6 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,symbol: input_.symbol,code: input_.code
  }!;
}export function jsonListCostBasesParamsFilterToTransportTransform(
  input_?: ListCostBasesParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    fiat_code: input_.fiatCode
  }!;
}export function jsonListCostBasesParamsFilterToApplicationTransform(
  input_?: any,
): ListCostBasesParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    fiatCode: input_.fiat_code
  }!;
}export function jsonArrayCostBasisToTransportTransform(
  items_?: Array<CostBasis> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonCostBasisToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayCostBasisToApplicationTransform(
  items_?: any,
): Array<CostBasis> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonCostBasisToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonCostBasisToTransportTransform(
  input_?: CostBasis | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,fiat_code: input_.fiatCode,rate: input_.rate,effective_from: dateRfc3339Serializer(input_.effectiveFrom),created_at: dateRfc3339Serializer(input_.createdAt)
  }!;
}export function jsonCostBasisToApplicationTransform(input_?: any): CostBasis {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,fiatCode: input_.fiat_code,rate: input_.rate,effectiveFrom: dateDeserializer(input_.effective_from)!,createdAt: dateDeserializer(input_.created_at)!
  }!;
}export function jsonCreateRequestToTransportTransform_7(
  input_?: CreateRequest_5 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    fiat_code: input_.fiatCode,rate: input_.rate,effective_from: dateRfc3339Serializer(input_.effectiveFrom)
  }!;
}export function jsonCreateRequestToApplicationTransform_7(
  input_?: any,
): CreateRequest_5 {
  if(!input_) {
    return input_ as any;
  }
    return {
    fiatCode: input_.fiat_code,rate: input_.rate,effectiveFrom: dateDeserializer(input_.effective_from)!
  }!;
}export function jsonListFeaturesParamsFilterToTransportTransform(
  input_?: ListFeaturesParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    meter_id: jsonUlidFieldFilterToTransportTransform(input_.meterId),key: jsonStringFieldFilterToTransportTransform(input_.key),name: jsonStringFieldFilterToTransportTransform(input_.name)
  }!;
}export function jsonListFeaturesParamsFilterToApplicationTransform(
  input_?: any,
): ListFeaturesParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    meterId: jsonUlidFieldFilterToApplicationTransform(input_.meter_id),key: jsonStringFieldFilterToApplicationTransform(input_.key),name: jsonStringFieldFilterToApplicationTransform(input_.name)
  }!;
}export function jsonArrayFeatureToTransportTransform(
  items_?: Array<Feature> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonFeatureToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayFeatureToApplicationTransform(
  items_?: any,
): Array<Feature> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonFeatureToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonFeatureToTransportTransform(input_?: Feature | null): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),key: input_.key,meter: jsonFeatureMeterReferenceToTransportTransform(input_.meter),unit_cost: jsonFeatureUnitCostToTransportTransform(input_.unitCost)
  }!;
}export function jsonFeatureToApplicationTransform(input_?: any): Feature {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,key: input_.key,meter: jsonFeatureMeterReferenceToApplicationTransform(input_.meter),unitCost: jsonFeatureUnitCostToApplicationTransform(input_.unit_cost)
  }!;
}export function jsonFeatureMeterReferenceToTransportTransform(
  input_?: FeatureMeterReference | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,filters: jsonRecordQueryFilterStringMapItemToTransportTransform(input_.filters)
  }!;
}export function jsonFeatureMeterReferenceToApplicationTransform(
  input_?: any,
): FeatureMeterReference {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,filters: jsonRecordQueryFilterStringMapItemToApplicationTransform(input_.filters)
  }!;
}export function jsonFeatureUnitCostToTransportDiscriminator(
  input_?: FeatureUnitCost,
): any {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.type;if( discriminatorValue === "manual") {
    return jsonFeatureManualUnitCostToTransportTransform(input_ as any)!
  }

  if( discriminatorValue === "llm") {
    return jsonFeatureLlmUnitCostToTransportTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonFeatureUnitCostToTransportTransform(
  input_?: FeatureUnitCost | null,
): any {
  if(!input_) {
    return input_ as any;
  }return jsonFeatureUnitCostToTransportDiscriminator(input_)
}export function jsonFeatureUnitCostToApplicationDiscriminator(
  input_?: any,
): FeatureUnitCost {
  if(!input_) {
    return input_ as any;
  }const discriminatorValue = input_.type;if( discriminatorValue === "manual") {
    return jsonFeatureManualUnitCostToApplicationTransform(input_ as any)!
  }

  if( discriminatorValue === "llm") {
    return jsonFeatureLlmUnitCostToApplicationTransform(input_ as any)!
  }console.warn(`Received unknown kind: ` + discriminatorValue); return input_ as any
}export function jsonFeatureUnitCostToApplicationTransform(
  input_?: any,
): FeatureUnitCost {
  if(!input_) {
    return input_ as any;
  }return jsonFeatureUnitCostToApplicationDiscriminator(input_)
}export function jsonFeatureManualUnitCostToTransportTransform(
  input_?: FeatureManualUnitCost | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,amount: input_.amount
  }!;
}export function jsonFeatureManualUnitCostToApplicationTransform(
  input_?: any,
): FeatureManualUnitCost {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,amount: input_.amount
  }!;
}export function jsonFeatureLlmUnitCostToTransportTransform(
  input_?: FeatureLlmUnitCost | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,provider_property: input_.providerProperty,provider: input_.provider,model_property: input_.modelProperty,model: input_.model,token_type_property: input_.tokenTypeProperty,token_type: input_.tokenType,pricing: jsonFeatureLlmUnitCostPricingToTransportTransform(input_.pricing)
  }!;
}export function jsonFeatureLlmUnitCostToApplicationTransform(
  input_?: any,
): FeatureLlmUnitCost {
  if(!input_) {
    return input_ as any;
  }
    return {
    type: input_.type,providerProperty: input_.provider_property,provider: input_.provider,modelProperty: input_.model_property,model: input_.model,tokenTypeProperty: input_.token_type_property,tokenType: input_.token_type,pricing: jsonFeatureLlmUnitCostPricingToApplicationTransform(input_.pricing)
  }!;
}export function jsonFeatureLlmUnitCostPricingToTransportTransform(
  input_?: FeatureLlmUnitCostPricing | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    input_per_token: input_.inputPerToken,output_per_token: input_.outputPerToken,cache_read_per_token: input_.cacheReadPerToken,reasoning_per_token: input_.reasoningPerToken,cache_write_per_token: input_.cacheWritePerToken
  }!;
}export function jsonFeatureLlmUnitCostPricingToApplicationTransform(
  input_?: any,
): FeatureLlmUnitCostPricing {
  if(!input_) {
    return input_ as any;
  }
    return {
    inputPerToken: input_.input_per_token,outputPerToken: input_.output_per_token,cacheReadPerToken: input_.cache_read_per_token,reasoningPerToken: input_.reasoning_per_token,cacheWritePerToken: input_.cache_write_per_token
  }!;
}export function jsonCreateRequestToTransportTransform_8(
  input_?: CreateRequest_4 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),key: input_.key,meter: jsonFeatureMeterReferenceToTransportTransform(input_.meter),unit_cost: jsonFeatureUnitCostToTransportTransform(input_.unitCost)
  }!;
}export function jsonCreateRequestToApplicationTransform_8(
  input_?: any,
): CreateRequest_4 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),key: input_.key,meter: jsonFeatureMeterReferenceToApplicationTransform(input_.meter),unitCost: jsonFeatureUnitCostToApplicationTransform(input_.unit_cost)
  }!;
}export function jsonFeatureUpdateRequestToTransportTransform(
  input_?: FeatureUpdateRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    unit_cost: jsonFeatureUnitCostToTransportTransform(input_.unitCost)
  }!;
}export function jsonFeatureUpdateRequestToApplicationTransform(
  input_?: any,
): FeatureUpdateRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    unitCost: jsonFeatureUnitCostToApplicationTransform(input_.unit_cost)
  }!;
}export function jsonFeatureCostQueryResultToTransportTransform(
  input_?: FeatureCostQueryResult | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    from: dateRfc3339Serializer(input_.from),to: dateRfc3339Serializer(input_.to),data: jsonArrayFeatureCostQueryRowToTransportTransform(input_.data)
  }!;
}export function jsonFeatureCostQueryResultToApplicationTransform(
  input_?: any,
): FeatureCostQueryResult {
  if(!input_) {
    return input_ as any;
  }
    return {
    from: dateDeserializer(input_.from)!,to: dateDeserializer(input_.to)!,data: jsonArrayFeatureCostQueryRowToApplicationTransform(input_.data)
  }!;
}export function jsonArrayFeatureCostQueryRowToTransportTransform(
  items_?: Array<FeatureCostQueryRow> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonFeatureCostQueryRowToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayFeatureCostQueryRowToApplicationTransform(
  items_?: any,
): Array<FeatureCostQueryRow> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonFeatureCostQueryRowToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonFeatureCostQueryRowToTransportTransform(
  input_?: FeatureCostQueryRow | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    usage: input_.usage,cost: input_.cost,currency: input_.currency,detail: input_.detail,from: dateRfc3339Serializer(input_.from),to: dateRfc3339Serializer(input_.to),dimensions: jsonRecordStringToTransportTransform(input_.dimensions)
  }!;
}export function jsonFeatureCostQueryRowToApplicationTransform(
  input_?: any,
): FeatureCostQueryRow {
  if(!input_) {
    return input_ as any;
  }
    return {
    usage: input_.usage,cost: input_.cost,currency: input_.currency,detail: input_.detail,from: dateDeserializer(input_.from)!,to: dateDeserializer(input_.to)!,dimensions: jsonRecordStringToApplicationTransform(input_.dimensions)
  }!;
}export function jsonListPricesParamsFilterToTransportTransform(
  input_?: ListPricesParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    provider: jsonStringFieldFilterToTransportTransform(input_.provider),model_id: jsonStringFieldFilterToTransportTransform(input_.modelId),model_name: jsonStringFieldFilterToTransportTransform(input_.modelName),currency: jsonStringFieldFilterToTransportTransform(input_.currency),source: jsonStringFieldFilterToTransportTransform(input_.source)
  }!;
}export function jsonListPricesParamsFilterToApplicationTransform(
  input_?: any,
): ListPricesParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    provider: jsonStringFieldFilterToApplicationTransform(input_.provider),modelId: jsonStringFieldFilterToApplicationTransform(input_.model_id),modelName: jsonStringFieldFilterToApplicationTransform(input_.model_name),currency: jsonStringFieldFilterToApplicationTransform(input_.currency),source: jsonStringFieldFilterToApplicationTransform(input_.source)
  }!;
}export function jsonArrayPriceToTransportTransform(
  items_?: Array<Price_2> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonPriceToTransportTransform_2(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayPriceToApplicationTransform(
  items_?: any,
): Array<Price_2> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonPriceToApplicationTransform_2(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonPriceToTransportTransform_2(input_?: Price_2 | null): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,provider: jsonProviderToTransportTransform(input_.provider),model: jsonModelToTransportTransform(input_.model),pricing: jsonModelPricingToTransportTransform(input_.pricing),currency: input_.currency,source: input_.source,effective_from: dateRfc3339Serializer(input_.effectiveFrom),effective_to: dateRfc3339Serializer(input_.effectiveTo),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt)
  }!;
}export function jsonPriceToApplicationTransform_2(input_?: any): Price_2 {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,provider: jsonProviderToApplicationTransform(input_.provider),model: jsonModelToApplicationTransform(input_.model),pricing: jsonModelPricingToApplicationTransform(input_.pricing),currency: input_.currency,source: input_.source,effectiveFrom: dateDeserializer(input_.effective_from)!,effectiveTo: dateDeserializer(input_.effective_to)!,createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!
  }!;
}export function jsonProviderToTransportTransform(
  input_?: Provider | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name
  }!;
}export function jsonProviderToApplicationTransform(input_?: any): Provider {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name
  }!;
}export function jsonModelToTransportTransform(input_?: Model | null): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name
  }!;
}export function jsonModelToApplicationTransform(input_?: any): Model {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name
  }!;
}export function jsonModelPricingToTransportTransform(
  input_?: ModelPricing | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    input_per_token: input_.inputPerToken,output_per_token: input_.outputPerToken,cache_read_per_token: input_.cacheReadPerToken,cache_write_per_token: input_.cacheWritePerToken,reasoning_per_token: input_.reasoningPerToken
  }!;
}export function jsonModelPricingToApplicationTransform(
  input_?: any,
): ModelPricing {
  if(!input_) {
    return input_ as any;
  }
    return {
    inputPerToken: input_.input_per_token,outputPerToken: input_.output_per_token,cacheReadPerToken: input_.cache_read_per_token,cacheWritePerToken: input_.cache_write_per_token,reasoningPerToken: input_.reasoning_per_token
  }!;
}export function jsonOverrideCreateToTransportTransform(
  input_?: OverrideCreate | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    provider: input_.provider,model_id: input_.modelId,model_name: input_.modelName,pricing: jsonModelPricingToTransportTransform(input_.pricing),currency: input_.currency,effective_from: dateRfc3339Serializer(input_.effectiveFrom),effective_to: dateRfc3339Serializer(input_.effectiveTo)
  }!;
}export function jsonOverrideCreateToApplicationTransform(
  input_?: any,
): OverrideCreate {
  if(!input_) {
    return input_ as any;
  }
    return {
    provider: input_.provider,modelId: input_.model_id,modelName: input_.model_name,pricing: jsonModelPricingToApplicationTransform(input_.pricing),currency: input_.currency,effectiveFrom: dateDeserializer(input_.effective_from)!,effectiveTo: dateDeserializer(input_.effective_to)!
  }!;
}export function jsonListPlansParamsFilterToTransportTransform(
  input_?: ListPlansParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    key: jsonStringFieldFilterToTransportTransform(input_.key),name: jsonStringFieldFilterToTransportTransform(input_.name),status: jsonStringFieldFilterExactToTransportTransform(input_.status),currency: jsonStringFieldFilterExactToTransportTransform(input_.currency)
  }!;
}export function jsonListPlansParamsFilterToApplicationTransform(
  input_?: any,
): ListPlansParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    key: jsonStringFieldFilterToApplicationTransform(input_.key),name: jsonStringFieldFilterToApplicationTransform(input_.name),status: jsonStringFieldFilterExactToApplicationTransform(input_.status),currency: jsonStringFieldFilterExactToApplicationTransform(input_.currency)
  }!;
}export function jsonArrayPlanToTransportTransform(
  items_?: Array<Plan> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonPlanToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayPlanToApplicationTransform(
  items_?: any,
): Array<Plan> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonPlanToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonPlanToTransportTransform(input_?: Plan | null): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),key: input_.key,version: input_.version,currency: input_.currency,billing_cadence: input_.billingCadence,pro_rating_enabled: input_.proRatingEnabled,effective_from: dateRfc3339Serializer(input_.effectiveFrom),effective_to: dateRfc3339Serializer(input_.effectiveTo),status: input_.status,phases: jsonArrayPlanPhaseToTransportTransform(input_.phases),validation_errors: jsonArrayProductCatalogValidationErrorToTransportTransform(input_.validationErrors)
  }!;
}export function jsonPlanToApplicationTransform(input_?: any): Plan {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,key: input_.key,version: input_.version,currency: input_.currency,billingCadence: input_.billing_cadence,proRatingEnabled: input_.pro_rating_enabled,effectiveFrom: dateDeserializer(input_.effective_from)!,effectiveTo: dateDeserializer(input_.effective_to)!,status: input_.status,phases: jsonArrayPlanPhaseToApplicationTransform(input_.phases),validationErrors: jsonArrayProductCatalogValidationErrorToApplicationTransform(input_.validation_errors)
  }!;
}export function jsonArrayPlanPhaseToTransportTransform(
  items_?: Array<PlanPhase> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonPlanPhaseToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayPlanPhaseToApplicationTransform(
  items_?: any,
): Array<PlanPhase> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonPlanPhaseToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonPlanPhaseToTransportTransform(
  input_?: PlanPhase | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),key: input_.key,duration: input_.duration,rate_cards: jsonArrayRateCardToTransportTransform(input_.rateCards)
  }!;
}export function jsonPlanPhaseToApplicationTransform(input_?: any): PlanPhase {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),key: input_.key,duration: input_.duration,rateCards: jsonArrayRateCardToApplicationTransform(input_.rate_cards)
  }!;
}export function jsonArrayRateCardToTransportTransform(
  items_?: Array<RateCard> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonRateCardToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayRateCardToApplicationTransform(
  items_?: any,
): Array<RateCard> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonRateCardToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonRateCardToTransportTransform(
  input_?: RateCard | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),key: input_.key,feature: jsonResourceReferenceToTransportTransform_4(input_.feature),billing_cadence: input_.billingCadence,price: jsonPriceToTransportTransform(input_.price),payment_term: jsonPricePaymentTermToTransportTransform(input_.paymentTerm),commitments: jsonSpendCommitmentsToTransportTransform(input_.commitments),discounts: jsonDiscountsToTransportTransform(input_.discounts),tax_config: jsonRateCardTaxConfigToTransportTransform(input_.taxConfig)
  }!;
}export function jsonRateCardToApplicationTransform(input_?: any): RateCard {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),key: input_.key,feature: jsonResourceReferenceToApplicationTransform_4(input_.feature),billingCadence: input_.billing_cadence,price: jsonPriceToApplicationTransform(input_.price),paymentTerm: jsonPricePaymentTermToApplicationTransform(input_.payment_term),commitments: jsonSpendCommitmentsToApplicationTransform(input_.commitments),discounts: jsonDiscountsToApplicationTransform(input_.discounts),taxConfig: jsonRateCardTaxConfigToApplicationTransform(input_.tax_config)
  }!;
}export function jsonResourceReferenceToTransportTransform_4(
  input_?: ResourceReference_4 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonResourceReferenceToApplicationTransform_4(
  input_?: any,
): ResourceReference_4 {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id
  }!;
}export function jsonSpendCommitmentsToTransportTransform(
  input_?: SpendCommitments | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    minimum_amount: input_.minimumAmount,maximum_amount: input_.maximumAmount
  }!;
}export function jsonSpendCommitmentsToApplicationTransform(
  input_?: any,
): SpendCommitments {
  if(!input_) {
    return input_ as any;
  }
    return {
    minimumAmount: input_.minimum_amount,maximumAmount: input_.maximum_amount
  }!;
}export function jsonRateCardTaxConfigToTransportTransform(
  input_?: RateCardTaxConfig | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    behavior: input_.behavior,code: jsonResourceReferenceToTransportTransform_2(input_.code)
  }!;
}export function jsonRateCardTaxConfigToApplicationTransform(
  input_?: any,
): RateCardTaxConfig {
  if(!input_) {
    return input_ as any;
  }
    return {
    behavior: input_.behavior,code: jsonResourceReferenceToApplicationTransform_2(input_.code)
  }!;
}export function jsonArrayProductCatalogValidationErrorToTransportTransform(
  items_?: Array<ProductCatalogValidationError> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonProductCatalogValidationErrorToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayProductCatalogValidationErrorToApplicationTransform(
  items_?: any,
): Array<ProductCatalogValidationError> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonProductCatalogValidationErrorToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonProductCatalogValidationErrorToTransportTransform(
  input_?: ProductCatalogValidationError | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code,message: input_.message,attributes: jsonRecordUnknownToTransportTransform(input_.attributes),field: input_.field
  }!;
}export function jsonProductCatalogValidationErrorToApplicationTransform(
  input_?: any,
): ProductCatalogValidationError {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code,message: input_.message,attributes: jsonRecordUnknownToApplicationTransform(input_.attributes),field: input_.field
  }!;
}export function jsonCreateRequestToTransportTransform_9(
  input_?: CreateRequest_3 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),key: input_.key,currency: input_.currency,billing_cadence: input_.billingCadence,pro_rating_enabled: input_.proRatingEnabled,phases: jsonArrayPlanPhaseToTransportTransform(input_.phases)
  }!;
}export function jsonCreateRequestToApplicationTransform_9(
  input_?: any,
): CreateRequest_3 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),key: input_.key,currency: input_.currency,billingCadence: input_.billing_cadence,proRatingEnabled: input_.pro_rating_enabled,phases: jsonArrayPlanPhaseToApplicationTransform(input_.phases)
  }!;
}export function jsonUpsertRequestToTransportTransform_6(
  input_?: UpsertRequest_3 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),pro_rating_enabled: input_.proRatingEnabled,phases: jsonArrayPlanPhaseToTransportTransform(input_.phases)
  }!;
}export function jsonUpsertRequestToApplicationTransform_6(
  input_?: any,
): UpsertRequest_3 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),proRatingEnabled: input_.pro_rating_enabled,phases: jsonArrayPlanPhaseToApplicationTransform(input_.phases)
  }!;
}export function jsonListAddonsParamsFilterToTransportTransform(
  input_?: ListAddonsParamsFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: jsonUlidFieldFilterToTransportTransform(input_.id),key: jsonStringFieldFilterToTransportTransform(input_.key),name: jsonStringFieldFilterToTransportTransform(input_.name),status: jsonStringFieldFilterExactToTransportTransform(input_.status),currency: jsonStringFieldFilterExactToTransportTransform(input_.currency)
  }!;
}export function jsonListAddonsParamsFilterToApplicationTransform(
  input_?: any,
): ListAddonsParamsFilter {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: jsonUlidFieldFilterToApplicationTransform(input_.id),key: jsonStringFieldFilterToApplicationTransform(input_.key),name: jsonStringFieldFilterToApplicationTransform(input_.name),status: jsonStringFieldFilterExactToApplicationTransform(input_.status),currency: jsonStringFieldFilterExactToApplicationTransform(input_.currency)
  }!;
}export function jsonArrayAddonToTransportTransform(
  items_?: Array<Addon> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonAddonToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayAddonToApplicationTransform(
  items_?: any,
): Array<Addon> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonAddonToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonAddonToTransportTransform(input_?: Addon | null): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),key: input_.key,version: input_.version,instance_type: input_.instanceType,currency: jsonCurrencyCodeToTransportTransform(input_.currency),effective_from: dateRfc3339Serializer(input_.effectiveFrom),effective_to: dateRfc3339Serializer(input_.effectiveTo),status: input_.status,rate_cards: jsonArrayRateCardToTransportTransform(input_.rateCards),validation_errors: jsonArrayProductCatalogValidationErrorToTransportTransform(input_.validationErrors)
  }!;
}export function jsonAddonToApplicationTransform(input_?: any): Addon {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,key: input_.key,version: input_.version,instanceType: input_.instance_type,currency: jsonCurrencyCodeToApplicationTransform(input_.currency),effectiveFrom: dateDeserializer(input_.effective_from)!,effectiveTo: dateDeserializer(input_.effective_to)!,status: input_.status,rateCards: jsonArrayRateCardToApplicationTransform(input_.rate_cards),validationErrors: jsonArrayProductCatalogValidationErrorToApplicationTransform(input_.validation_errors)
  }!;
}export function jsonCreateRequestToTransportTransform_10(
  input_?: CreateRequest_2 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),key: input_.key,instance_type: input_.instanceType,currency: jsonCurrencyCodeToTransportTransform(input_.currency),rate_cards: jsonArrayRateCardToTransportTransform(input_.rateCards)
  }!;
}export function jsonCreateRequestToApplicationTransform_10(
  input_?: any,
): CreateRequest_2 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),key: input_.key,instanceType: input_.instance_type,currency: jsonCurrencyCodeToApplicationTransform(input_.currency),rateCards: jsonArrayRateCardToApplicationTransform(input_.rate_cards)
  }!;
}export function jsonUpsertRequestToTransportTransform_7(
  input_?: UpsertRequest_2 | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),instance_type: input_.instanceType,rate_cards: jsonArrayRateCardToTransportTransform(input_.rateCards)
  }!;
}export function jsonUpsertRequestToApplicationTransform_7(
  input_?: any,
): UpsertRequest_2 {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),instanceType: input_.instance_type,rateCards: jsonArrayRateCardToApplicationTransform(input_.rate_cards)
  }!;
}export function jsonArrayPlanAddonToTransportTransform(
  items_?: Array<PlanAddon> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonPlanAddonToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayPlanAddonToApplicationTransform(
  items_?: any,
): Array<PlanAddon> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonPlanAddonToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonPlanAddonToTransportTransform(
  input_?: PlanAddon | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt),deleted_at: dateRfc3339Serializer(input_.deletedAt),addon: jsonResourceReferenceToTransportTransform_3(input_.addon),from_plan_phase: input_.fromPlanPhase,max_quantity: input_.maxQuantity,validation_errors: jsonArrayProductCatalogValidationErrorToTransportTransform(input_.validationErrors)
  }!;
}export function jsonPlanAddonToApplicationTransform(input_?: any): PlanAddon {
  if(!input_) {
    return input_ as any;
  }
    return {
    id: input_.id,name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!,deletedAt: dateDeserializer(input_.deleted_at)!,addon: jsonResourceReferenceToApplicationTransform_3(input_.addon),fromPlanPhase: input_.from_plan_phase,maxQuantity: input_.max_quantity,validationErrors: jsonArrayProductCatalogValidationErrorToApplicationTransform(input_.validation_errors)
  }!;
}export function jsonCreateRequestToTransportTransform_11(
  input_?: CreateRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),addon: jsonResourceReferenceToTransportTransform_3(input_.addon),from_plan_phase: input_.fromPlanPhase,max_quantity: input_.maxQuantity
  }!;
}export function jsonCreateRequestToApplicationTransform_11(
  input_?: any,
): CreateRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),addon: jsonResourceReferenceToApplicationTransform_3(input_.addon),fromPlanPhase: input_.from_plan_phase,maxQuantity: input_.max_quantity
  }!;
}export function jsonUpsertRequestToTransportTransform_8(
  input_?: UpsertRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToTransportTransform(input_.labels),from_plan_phase: input_.fromPlanPhase,max_quantity: input_.maxQuantity
  }!;
}export function jsonUpsertRequestToApplicationTransform_8(
  input_?: any,
): UpsertRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    name: input_.name,description: input_.description,labels: jsonLabelsToApplicationTransform(input_.labels),fromPlanPhase: input_.from_plan_phase,maxQuantity: input_.max_quantity
  }!;
}export function jsonOrganizationDefaultTaxCodesToTransportTransform(
  input_?: OrganizationDefaultTaxCodes | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    invoicing_tax_code: jsonResourceReferenceToTransportTransform_2(input_.invoicingTaxCode),credit_grant_tax_code: jsonResourceReferenceToTransportTransform_2(input_.creditGrantTaxCode),created_at: dateRfc3339Serializer(input_.createdAt),updated_at: dateRfc3339Serializer(input_.updatedAt)
  }!;
}export function jsonOrganizationDefaultTaxCodesToApplicationTransform(
  input_?: any,
): OrganizationDefaultTaxCodes {
  if(!input_) {
    return input_ as any;
  }
    return {
    invoicingTaxCode: jsonResourceReferenceToApplicationTransform_2(input_.invoicing_tax_code),creditGrantTaxCode: jsonResourceReferenceToApplicationTransform_2(input_.credit_grant_tax_code),createdAt: dateDeserializer(input_.created_at)!,updatedAt: dateDeserializer(input_.updated_at)!
  }!;
}export function jsonUpdateRequestToTransportTransform_2(
  input_?: UpdateRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    invoicing_tax_code: jsonResourceReferenceToTransportTransform_2(input_.invoicingTaxCode),credit_grant_tax_code: jsonResourceReferenceToTransportTransform_2(input_.creditGrantTaxCode)
  }!;
}export function jsonUpdateRequestToApplicationTransform_2(
  input_?: any,
): UpdateRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    invoicingTaxCode: jsonResourceReferenceToApplicationTransform_2(input_.invoicing_tax_code),creditGrantTaxCode: jsonResourceReferenceToApplicationTransform_2(input_.credit_grant_tax_code)
  }!;
}export function jsonGovernanceQueryRequestToTransportTransform(
  input_?: GovernanceQueryRequest | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    include_credits: input_.includeCredits,customer: jsonGovernanceQueryRequestCustomersToTransportTransform(input_.customer),feature: jsonGovernanceQueryRequestFeaturesToTransportTransform(input_.feature)
  }!;
}export function jsonGovernanceQueryRequestToApplicationTransform(
  input_?: any,
): GovernanceQueryRequest {
  if(!input_) {
    return input_ as any;
  }
    return {
    includeCredits: input_.include_credits,customer: jsonGovernanceQueryRequestCustomersToApplicationTransform(input_.customer),feature: jsonGovernanceQueryRequestFeaturesToApplicationTransform(input_.feature)
  }!;
}export function jsonGovernanceQueryRequestCustomersToTransportTransform(
  input_?: GovernanceQueryRequestCustomers | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    keys: jsonArrayStringToTransportTransform(input_.keys)
  }!;
}export function jsonGovernanceQueryRequestCustomersToApplicationTransform(
  input_?: any,
): GovernanceQueryRequestCustomers {
  if(!input_) {
    return input_ as any;
  }
    return {
    keys: jsonArrayStringToApplicationTransform(input_.keys)
  }!;
}export function jsonGovernanceQueryRequestFeaturesToTransportTransform(
  input_?: GovernanceQueryRequestFeatures | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    keys: jsonArrayStringToTransportTransform(input_.keys)
  }!;
}export function jsonGovernanceQueryRequestFeaturesToApplicationTransform(
  input_?: any,
): GovernanceQueryRequestFeatures {
  if(!input_) {
    return input_ as any;
  }
    return {
    keys: jsonArrayStringToApplicationTransform(input_.keys)
  }!;
}export function jsonGovernanceQueryResponseToTransportTransform(
  input_?: GovernanceQueryResponse | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    data: jsonArrayGovernanceQueryResultToTransportTransform(input_.data),errors: jsonArrayGovernanceQueryErrorToTransportTransform(input_.errors),meta: jsonCursorMetaToTransportTransform(input_.meta)
  }!;
}export function jsonGovernanceQueryResponseToApplicationTransform(
  input_?: any,
): GovernanceQueryResponse {
  if(!input_) {
    return input_ as any;
  }
    return {
    data: jsonArrayGovernanceQueryResultToApplicationTransform(input_.data),errors: jsonArrayGovernanceQueryErrorToApplicationTransform(input_.errors),meta: jsonCursorMetaToApplicationTransform(input_.meta)
  }!;
}export function jsonArrayGovernanceQueryResultToTransportTransform(
  items_?: Array<GovernanceQueryResult> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonGovernanceQueryResultToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayGovernanceQueryResultToApplicationTransform(
  items_?: any,
): Array<GovernanceQueryResult> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonGovernanceQueryResultToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonGovernanceQueryResultToTransportTransform(
  input_?: GovernanceQueryResult | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    matched: jsonArrayStringToTransportTransform(input_.matched),customer: jsonCustomerToTransportTransform(input_.customer),features: jsonRecordGovernanceFeatureAccessToTransportTransform(input_.features),updated_at: dateRfc3339Serializer(input_.updatedAt)
  }!;
}export function jsonGovernanceQueryResultToApplicationTransform(
  input_?: any,
): GovernanceQueryResult {
  if(!input_) {
    return input_ as any;
  }
    return {
    matched: jsonArrayStringToApplicationTransform(input_.matched),customer: jsonCustomerToApplicationTransform(input_.customer),features: jsonRecordGovernanceFeatureAccessToApplicationTransform(input_.features),updatedAt: dateDeserializer(input_.updated_at)!
  }!;
}export function jsonRecordGovernanceFeatureAccessToTransportTransform(
  items_?: Record<string, any> | null,
): any {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = jsonGovernanceFeatureAccessToTransportTransform(value as any);
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonRecordGovernanceFeatureAccessToApplicationTransform(
  items_?: any,
): Record<string, any> {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = jsonGovernanceFeatureAccessToApplicationTransform(value as any);
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonGovernanceFeatureAccessToTransportTransform(
  input_?: GovernanceFeatureAccess | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    has_access: input_.hasAccess,reason: jsonGovernanceFeatureAccessReasonToTransportTransform(input_.reason)
  }!;
}export function jsonGovernanceFeatureAccessToApplicationTransform(
  input_?: any,
): GovernanceFeatureAccess {
  if(!input_) {
    return input_ as any;
  }
    return {
    hasAccess: input_.has_access,reason: jsonGovernanceFeatureAccessReasonToApplicationTransform(input_.reason)
  }!;
}export function jsonGovernanceFeatureAccessReasonToTransportTransform(
  input_?: GovernanceFeatureAccessReason | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code,message: input_.message,attributes: jsonRecordUnknownToTransportTransform(input_.attributes)
  }!;
}export function jsonGovernanceFeatureAccessReasonToApplicationTransform(
  input_?: any,
): GovernanceFeatureAccessReason {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code,message: input_.message,attributes: jsonRecordUnknownToApplicationTransform(input_.attributes)
  }!;
}export function jsonArrayGovernanceQueryErrorToTransportTransform(
  items_?: Array<GovernanceQueryError> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonGovernanceQueryErrorToTransportTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayGovernanceQueryErrorToApplicationTransform(
  items_?: any,
): Array<GovernanceQueryError> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = jsonGovernanceQueryErrorToApplicationTransform(item as any);
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonGovernanceQueryErrorToTransportTransform(
  input_?: GovernanceQueryError | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code,message: input_.message,attributes: jsonRecordUnknownToTransportTransform(input_.attributes),customer: input_.customer
  }!;
}export function jsonGovernanceQueryErrorToApplicationTransform(
  input_?: any,
): GovernanceQueryError {
  if(!input_) {
    return input_ as any;
  }
    return {
    code: input_.code,message: input_.message,attributes: jsonRecordUnknownToApplicationTransform(input_.attributes),customer: input_.customer
  }!;
}export function jsonFieldFiltersToTransportTransform(
  input_?: FieldFilters | null,
): any {
  if(!input_) {
    return input_ as any;
  }
    return {
    boolean: jsonBooleanFieldFilterToTransportTransform(input_.boolean),numeric: jsonNumericFieldFilterToTransportTransform(input_.numeric),string: jsonStringFieldFilterToTransportTransform(input_.string),string_exact: jsonStringFieldFilterExactToTransportTransform(input_.stringExact),ulid: jsonUlidFieldFilterToTransportTransform(input_.ulid),datetime: jsonDateTimeFieldFilterToTransportTransform(input_.datetime),labels: jsonRecordStringFieldFilterToTransportTransform(input_.labels)
  }!;
}export function jsonFieldFiltersToApplicationTransform(
  input_?: any,
): FieldFilters {
  if(!input_) {
    return input_ as any;
  }
    return {
    boolean: jsonBooleanFieldFilterToApplicationTransform(input_.boolean),numeric: jsonNumericFieldFilterToApplicationTransform(input_.numeric),string: jsonStringFieldFilterToApplicationTransform(input_.string),stringExact: jsonStringFieldFilterExactToApplicationTransform(input_.string_exact),ulid: jsonUlidFieldFilterToApplicationTransform(input_.ulid),datetime: jsonDateTimeFieldFilterToApplicationTransform(input_.datetime),labels: jsonRecordStringFieldFilterToApplicationTransform(input_.labels)
  }!;
}export function jsonBooleanFieldFilterToTransportTransform(
  input_?: BooleanFieldFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonBooleanFieldFilterToApplicationTransform(
  input_?: any,
): BooleanFieldFilter {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonNumericFieldFilterToTransportTransform(
  input_?: NumericFieldFilter | null,
): any {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonNumericFieldFilterToApplicationTransform(
  input_?: any,
): NumericFieldFilter {
  if(!input_) {
    return input_ as any;
  }return input_
}export function jsonArrayFloat64ToTransportTransform(
  items_?: Array<number> | null,
): any {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonArrayFloat64ToApplicationTransform(
  items_?: any,
): Array<number> {
  if(!items_) {
    return items_ as any;
  }
  const _transformedArray = [];

  for (const item of items_ ?? []) {
    const transformedItem = item as any;
    _transformedArray.push(transformedItem);
  }

  return _transformedArray as any;
}export function jsonRecordStringFieldFilterToTransportTransform(
  items_?: Record<string, any> | null,
): any {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = jsonStringFieldFilterToTransportTransform(value as any);
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}export function jsonRecordStringFieldFilterToApplicationTransform(
  items_?: any,
): Record<string, any> {
  if(!items_) {
    return items_ as any;
  }

  const _transformedRecord: any = {};

  for (const [key, value] of Object.entries(items_ ?? {})) {
    const transformedItem = jsonStringFieldFilterToApplicationTransform(value as any);
    _transformedRecord[key] = transformedItem;
  }

  return _transformedRecord;
}
