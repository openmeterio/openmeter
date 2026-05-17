import {
  type AddonsEndpointsClientContext,
  type AddonsEndpointsClientOptions,
  createAddonsEndpointsClientContext,
} from "./api/addonsEndpointsClient/addonsEndpointsClientContext.js";
import {
  archiveAddon,
  type ArchiveAddonOptions,
  createAddon,
  type CreateAddonOptions,
  deleteAddon,
  type DeleteAddonOptions,
  getAddon,
  type GetAddonOptions,
  listAddons,
  type ListAddonsOptions,
  publishAddon,
  type PublishAddonOptions,
  updateAddon,
  type UpdateAddonOptions,
} from "./api/addonsEndpointsClient/addonsEndpointsClientOperations.js";
import {
  type AppsEndpointsClientContext,
  type AppsEndpointsClientOptions,
  createAppsEndpointsClientContext,
} from "./api/appsEndpointsClient/appsEndpointsClientContext.js";
import {
  get as get_7,
  type GetOptions as GetOptions_7,
  list as list_10,
  type ListOptions as ListOptions_10,
} from "./api/appsEndpointsClient/appsEndpointsClientOperations.js";
import {
  type BillingProfilesEndpointsClientContext,
  type BillingProfilesEndpointsClientOptions,
  createBillingProfilesEndpointsClientContext,
} from "./api/billingProfilesEndpointsClient/billingProfilesEndpointsClientContext.js";
import {
  create as create_6,
  type CreateOptions as CreateOptions_6,
  delete_ as delete__3,
  type DeleteOptions as DeleteOptions_3,
  get as get_8,
  type GetOptions as GetOptions_8,
  list as list_11,
  type ListOptions as ListOptions_11,
  update as update_2,
  type UpdateOptions as UpdateOptions_2,
} from "./api/billingProfilesEndpointsClient/billingProfilesEndpointsClientOperations.js";
import {
  createCurrenciesCustomCostBasesEndpointsClientContext,
  type CurrenciesCustomCostBasesEndpointsClientContext,
  type CurrenciesCustomCostBasesEndpointsClientOptions,
} from "./api/currenciesCustomCostBasesEndpointsClient/currenciesCustomCostBasesEndpointsClientContext.js";
import {
  createCostBasis,
  type CreateCostBasisOptions,
  getCostBases,
  type GetCostBasesOptions,
} from "./api/currenciesCustomCostBasesEndpointsClient/currenciesCustomCostBasesEndpointsClientOperations.js";
import {
  createCurrenciesCustomEndpointsClientContext,
  type CurrenciesCustomEndpointsClientContext,
  type CurrenciesCustomEndpointsClientOptions,
} from "./api/currenciesCustomEndpointsClient/currenciesCustomEndpointsClientContext.js";
import {
  create as create_8,
  type CreateOptions as CreateOptions_8,
} from "./api/currenciesCustomEndpointsClient/currenciesCustomEndpointsClientOperations.js";
import {
  createCurrenciesEndpointsClientContext,
  type CurrenciesEndpointsClientContext,
  type CurrenciesEndpointsClientOptions,
} from "./api/currenciesEndpointsClient/currenciesEndpointsClientContext.js";
import {
  list as list_13,
  type ListOptions as ListOptions_13,
} from "./api/currenciesEndpointsClient/currenciesEndpointsClientOperations.js";
import {
  createCustomerBillingEndpointsClientContext,
  type CustomerBillingEndpointsClientContext,
  type CustomerBillingEndpointsClientOptions,
} from "./api/customerBillingEndpointsClient/customerBillingEndpointsClientContext.js";
import {
  createCheckoutSession,
  type CreateCheckoutSessionOptions,
  createPortalSession,
  type CreatePortalSessionOptions,
  get as get_3,
  type GetOptions as GetOptions_3,
  upsert as upsert_2,
  upsertAppData,
  type UpsertAppDataOptions,
  type UpsertOptions as UpsertOptions_2,
} from "./api/customerBillingEndpointsClient/customerBillingEndpointsClientOperations.js";
import {
  createCustomerChargesEndpointsClientContext,
  type CustomerChargesEndpointsClientContext,
  type CustomerChargesEndpointsClientOptions,
} from "./api/customerChargesEndpointsClient/customerChargesEndpointsClientContext.js";
import {
  list as list_7,
  type ListOptions as ListOptions_7,
} from "./api/customerChargesEndpointsClient/customerChargesEndpointsClientOperations.js";
import {
  createCustomerCreditAdjustmentsEndpointsClientContext,
  type CustomerCreditAdjustmentsEndpointsClientContext,
  type CustomerCreditAdjustmentsEndpointsClientOptions,
} from "./api/customerCreditAdjustmentsEndpointsClient/customerCreditAdjustmentsEndpointsClientContext.js";
import {
  create as create_4,
  type CreateOptions as CreateOptions_4,
} from "./api/customerCreditAdjustmentsEndpointsClient/customerCreditAdjustmentsEndpointsClientOperations.js";
import {
  createCustomerCreditBalanceEndpointsClientContext,
  type CustomerCreditBalanceEndpointsClientContext,
  type CustomerCreditBalanceEndpointsClientOptions,
} from "./api/customerCreditBalanceEndpointsClient/customerCreditBalanceEndpointsClientContext.js";
import {
  get as get_5,
  type GetOptions as GetOptions_5,
} from "./api/customerCreditBalanceEndpointsClient/customerCreditBalanceEndpointsClientOperations.js";
import {
  createCustomerCreditGrantEndpointsClientContext,
  type CustomerCreditGrantEndpointsClientContext,
  type CustomerCreditGrantEndpointsClientOptions,
} from "./api/customerCreditGrantEndpointsClient/customerCreditGrantEndpointsClientContext.js";
import {
  updateExternalSettlement,
  type UpdateExternalSettlementOptions,
} from "./api/customerCreditGrantEndpointsClient/customerCreditGrantEndpointsClientOperations.js";
import {
  createCustomerCreditGrantsEndpointsClientContext,
  type CustomerCreditGrantsEndpointsClientContext,
  type CustomerCreditGrantsEndpointsClientOptions,
} from "./api/customerCreditGrantsEndpointsClient/customerCreditGrantsEndpointsClientContext.js";
import {
  create as create_3,
  type CreateOptions as CreateOptions_3,
  get as get_4,
  type GetOptions as GetOptions_4,
  list as list_5,
  type ListOptions as ListOptions_5,
} from "./api/customerCreditGrantsEndpointsClient/customerCreditGrantsEndpointsClientOperations.js";
import {
  createCustomerCreditTransactionEndpointsClientContext,
  type CustomerCreditTransactionEndpointsClientContext,
  type CustomerCreditTransactionEndpointsClientOptions,
} from "./api/customerCreditTransactionEndpointsClient/customerCreditTransactionEndpointsClientContext.js";
import {
  list as list_6,
  type ListOptions as ListOptions_6,
} from "./api/customerCreditTransactionEndpointsClient/customerCreditTransactionEndpointsClientOperations.js";
import {
  createCustomerEntitlementsEndpointsClientContext,
  type CustomerEntitlementsEndpointsClientContext,
  type CustomerEntitlementsEndpointsClientOptions,
} from "./api/customerEntitlementsEndpointsClient/customerEntitlementsEndpointsClientContext.js";
import {
  list as list_4,
  type ListOptions as ListOptions_4,
} from "./api/customerEntitlementsEndpointsClient/customerEntitlementsEndpointsClientOperations.js";
import {
  createCustomersEndpointsClientContext,
  type CustomersEndpointsClientContext,
  type CustomersEndpointsClientOptions,
} from "./api/customersEndpointsClient/customersEndpointsClientContext.js";
import {
  create as create_2,
  type CreateOptions as CreateOptions_2,
  delete_ as delete__2,
  type DeleteOptions as DeleteOptions_2,
  get as get_2,
  type GetOptions as GetOptions_2,
  list as list_3,
  type ListOptions as ListOptions_3,
  upsert,
  type UpsertOptions,
} from "./api/customersEndpointsClient/customersEndpointsClientOperations.js";
import {
  createEventsEndpointsClientContext,
  type EventsEndpointsClientContext,
  type EventsEndpointsClientOptions,
} from "./api/eventsEndpointsClient/eventsEndpointsClientContext.js";
import {
  ingestEvent,
  type IngestEventOptions,
  ingestEvents,
  ingestEventsJson,
  type IngestEventsJsonOptions,
  type IngestEventsOptions,
  list,
  type ListOptions,
} from "./api/eventsEndpointsClient/eventsEndpointsClientOperations.js";
import {
  createFeatureCostEndpointsClientContext,
  type FeatureCostEndpointsClientContext,
  type FeatureCostEndpointsClientOptions,
} from "./api/featureCostEndpointsClient/featureCostEndpointsClientContext.js";
import {
  queryCost,
  type QueryCostOptions,
} from "./api/featureCostEndpointsClient/featureCostEndpointsClientOperations.js";
import {
  createFeaturesEndpointsClientContext,
  type FeaturesEndpointsClientContext,
  type FeaturesEndpointsClientOptions,
} from "./api/featuresEndpointsClient/featuresEndpointsClientContext.js";
import {
  create as create_9,
  type CreateOptions as CreateOptions_9,
  delete_ as delete__5,
  type DeleteOptions as DeleteOptions_5,
  get as get_10,
  type GetOptions as GetOptions_10,
  list as list_14,
  type ListOptions as ListOptions_14,
  update as update_3,
  type UpdateOptions as UpdateOptions_3,
} from "./api/featuresEndpointsClient/featuresEndpointsClientOperations.js";
import {
  createGovernanceEndpointsClientContext,
  type GovernanceEndpointsClientContext,
  type GovernanceEndpointsClientOptions,
} from "./api/governanceEndpointsClient/governanceEndpointsClientContext.js";
import {
  query as query_2,
  type QueryOptions as QueryOptions_2,
} from "./api/governanceEndpointsClient/governanceEndpointsClientOperations.js";
import {
  createLlmCostOverridesEndpointsClientContext,
  type LlmCostOverridesEndpointsClientContext,
  type LlmCostOverridesEndpointsClientOptions,
} from "./api/llmCostOverridesEndpointsClient/llmCostOverridesEndpointsClientContext.js";
import {
  createOverride,
  type CreateOverrideOptions,
  deleteOverride,
  type DeleteOverrideOptions,
  listOverrides,
  type ListOverridesOptions,
} from "./api/llmCostOverridesEndpointsClient/llmCostOverridesEndpointsClientOperations.js";
import {
  createLlmCostPricesEndpointsClientContext,
  type LlmCostPricesEndpointsClientContext,
  type LlmCostPricesEndpointsClientOptions,
} from "./api/llmCostPricesEndpointsClient/llmCostPricesEndpointsClientContext.js";
import {
  getPrice,
  type GetPriceOptions,
  listPrices,
  type ListPricesOptions,
} from "./api/llmCostPricesEndpointsClient/llmCostPricesEndpointsClientOperations.js";
import {
  createMetersEndpointsClientContext,
  type MetersEndpointsClientContext,
  type MetersEndpointsClientOptions,
} from "./api/metersEndpointsClient/metersEndpointsClientContext.js";
import {
  create,
  type CreateOptions,
  delete_,
  type DeleteOptions,
  get,
  type GetOptions,
  list as list_2,
  type ListOptions as ListOptions_2,
  update,
  type UpdateOptions,
} from "./api/metersEndpointsClient/metersEndpointsClientOperations.js";
import {
  createMetersQueryEndpointsClientContext,
  type MetersQueryEndpointsClientContext,
  type MetersQueryEndpointsClientOptions,
} from "./api/metersQueryEndpointsClient/metersQueryEndpointsClientContext.js";
import {
  query,
  queryCsv,
  type QueryCsvOptions,
  type QueryOptions,
} from "./api/metersQueryEndpointsClient/metersQueryEndpointsClientOperations.js";
import {
  createOpenMeterClientContext,
  type OpenMeterClientContext,
  type OpenMeterClientOptions,
} from "./api/openMeterClientContext.js";
import {
  createOrganizationDefaultTaxCodesEndpointsClientContext,
  type OrganizationDefaultTaxCodesEndpointsClientContext,
  type OrganizationDefaultTaxCodesEndpointsClientOptions,
} from "./api/organizationDefaultTaxCodesEndpointsClient/organizationDefaultTaxCodesEndpointsClientContext.js";
import {
  get as get_11,
  type GetOptions as GetOptions_11,
  update as update_4,
  type UpdateOptions as UpdateOptions_4,
} from "./api/organizationDefaultTaxCodesEndpointsClient/organizationDefaultTaxCodesEndpointsClientOperations.js";
import {
  createPlanAddonEndpointsClientContext,
  type PlanAddonEndpointsClientContext,
  type PlanAddonEndpointsClientOptions,
} from "./api/planAddonEndpointsClient/planAddonEndpointsClientContext.js";
import {
  createPlanAddon,
  type CreatePlanAddonOptions,
  deletePlanAddon,
  type DeletePlanAddonOptions,
  getPlanAddon,
  type GetPlanAddonOptions,
  listPlanAddons,
  type ListPlanAddonsOptions,
  updatePlanAddon,
  type UpdatePlanAddonOptions,
} from "./api/planAddonEndpointsClient/planAddonEndpointsClientOperations.js";
import {
  createPlansEndpointsClientContext,
  type PlansEndpointsClientContext,
  type PlansEndpointsClientOptions,
} from "./api/plansEndpointsClient/plansEndpointsClientContext.js";
import {
  archivePlan,
  type ArchivePlanOptions,
  createPlan,
  type CreatePlanOptions,
  deletePlan,
  type DeletePlanOptions,
  getPlan,
  type GetPlanOptions,
  listPlans,
  type ListPlansOptions,
  publishPlan,
  type PublishPlanOptions,
  updatePlan,
  type UpdatePlanOptions,
} from "./api/plansEndpointsClient/plansEndpointsClientOperations.js";
import {
  createSubscriptionAddonEndpointsClientContext,
  type SubscriptionAddonEndpointsClientContext,
  type SubscriptionAddonEndpointsClientOptions,
} from "./api/subscriptionAddonEndpointsClient/subscriptionAddonEndpointsClientContext.js";
import {
  list as list_9,
  type ListOptions as ListOptions_9,
} from "./api/subscriptionAddonEndpointsClient/subscriptionAddonEndpointsClientOperations.js";
import {
  createSubscriptionsEndpointsClientContext,
  type SubscriptionsEndpointsClientContext,
  type SubscriptionsEndpointsClientOptions,
} from "./api/subscriptionsEndpointsClient/subscriptionsEndpointsClientContext.js";
import {
  cancel,
  type CancelOptions,
  change,
  type ChangeOptions,
  create as create_5,
  type CreateOptions as CreateOptions_5,
  get as get_6,
  type GetOptions as GetOptions_6,
  list as list_8,
  type ListOptions as ListOptions_8,
  unscheduleCancelation,
  type UnscheduleCancelationOptions,
} from "./api/subscriptionsEndpointsClient/subscriptionsEndpointsClientOperations.js";
import {
  createTaxCodesEndpointsClientContext,
  type TaxCodesEndpointsClientContext,
  type TaxCodesEndpointsClientOptions,
} from "./api/taxCodesEndpointsClient/taxCodesEndpointsClientContext.js";
import {
  create as create_7,
  type CreateOptions as CreateOptions_7,
  delete_ as delete__4,
  type DeleteOptions as DeleteOptions_4,
  get as get_9,
  type GetOptions as GetOptions_9,
  list as list_12,
  type ListOptions as ListOptions_12,
  upsert as upsert_3,
  type UpsertOptions as UpsertOptions_3,
} from "./api/taxCodesEndpointsClient/taxCodesEndpointsClientOperations.js";
import type {
  CreateRequest,
  CreateRequest_10,
  CreateRequest_11,
  CreateRequest_2,
  CreateRequest_3,
  CreateRequest_4,
  CreateRequest_5,
  CreateRequest_6,
  CreateRequest_7,
  CreateRequest_8,
  CreateRequest_9,
  CreateRequestNested,
  CustomerBillingStripeCreateCheckoutSessionRequest,
  CustomerBillingStripeCreateCustomerPortalSessionRequest,
  FeatureUpdateRequest,
  GovernanceQueryRequest,
  MeteringEvent,
  MeterQueryRequest,
  OverrideCreate,
  SubscriptionCancel,
  SubscriptionChange,
  SubscriptionCreate,
  UpdateCreditGrantExternalSettlementRequest,
  UpdateRequest,
  UpdateRequest_2,
  UpsertRequest,
  UpsertRequest_2,
  UpsertRequest_3,
  UpsertRequest_4,
  UpsertRequest_5,
  UpsertRequest_6,
  UpsertRequest_7,
  UpsertRequest_8,
} from "./models/models.js";

export class OpenMeterClient {
  #context: OpenMeterClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: OpenMeterClientOptions,
  ) {
    this.#context = createOpenMeterClientContext(endpoint, options);

  }
}
export class GovernanceEndpointsClient {
  #context: GovernanceEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: GovernanceEndpointsClientOptions,
  ) {
    this.#context = createGovernanceEndpointsClientContext(endpoint, options);

  }
  async query(_: GovernanceQueryRequest, options?: QueryOptions_2) {
    return query_2(this.#context, _, options);
  }
}
export class OrganizationDefaultTaxCodesEndpointsClient {
  #context: OrganizationDefaultTaxCodesEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: OrganizationDefaultTaxCodesEndpointsClientOptions,
  ) {
    this.#context = createOrganizationDefaultTaxCodesEndpointsClientContext(
      endpoint,
      options
    );

  }
  async get(options?: GetOptions_11) {
    return get_11(this.#context, options);
  };
  async update(body: UpdateRequest_2, options?: UpdateOptions_4) {
    return update_4(this.#context, body, options);
  }
}
export class PlanAddonEndpointsClient {
  #context: PlanAddonEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: PlanAddonEndpointsClientOptions,
  ) {
    this.#context = createPlanAddonEndpointsClientContext(endpoint, options);

  }
  listPlanAddons(planId: string, options?: ListPlanAddonsOptions) {
    return listPlanAddons(this.#context, planId, options);
  };
  async createPlanAddon(
    planId: string,
    planAddon: CreateRequest_11,
    options?: CreatePlanAddonOptions,
  ) {
    return createPlanAddon(this.#context, planId, planAddon, options);
  };
  async getPlanAddon(
    planId: string,
    planAddonId: string,
    options?: GetPlanAddonOptions,
  ) {
    return getPlanAddon(this.#context, planId, planAddonId, options);
  };
  async updatePlanAddon(
    planId: string,
    planAddonId: string,
    planAddon: UpsertRequest_8,
    options?: UpdatePlanAddonOptions,
  ) {
    return updatePlanAddon(
      this.#context,
      planId,
      planAddonId,
      planAddon,
      options
    );
  };
  async deletePlanAddon(
    planId: string,
    planAddonId: string,
    options?: DeletePlanAddonOptions,
  ) {
    return deletePlanAddon(this.#context, planId, planAddonId, options);
  }
}
export class AddonsEndpointsClient {
  #context: AddonsEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: AddonsEndpointsClientOptions,
  ) {
    this.#context = createAddonsEndpointsClientContext(endpoint, options);

  }
  listAddons(options?: ListAddonsOptions) {
    return listAddons(this.#context, options);
  };
  async createAddon(addon: CreateRequest_10, options?: CreateAddonOptions) {
    return createAddon(this.#context, addon, options);
  };
  async updateAddon(
    addonId: string,
    addon: UpsertRequest_7,
    options?: UpdateAddonOptions,
  ) {
    return updateAddon(this.#context, addonId, addon, options);
  };
  async getAddon(addonId: string, options?: GetAddonOptions) {
    return getAddon(this.#context, addonId, options);
  };
  async deleteAddon(addonId: string, options?: DeleteAddonOptions) {
    return deleteAddon(this.#context, addonId, options);
  };
  async archiveAddon(addonId: string, options?: ArchiveAddonOptions) {
    return archiveAddon(this.#context, addonId, options);
  };
  async publishAddon(addonId: string, options?: PublishAddonOptions) {
    return publishAddon(this.#context, addonId, options);
  }
}
export class PlansEndpointsClient {
  #context: PlansEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: PlansEndpointsClientOptions,
  ) {
    this.#context = createPlansEndpointsClientContext(endpoint, options);

  }
  listPlans(options?: ListPlansOptions) {
    return listPlans(this.#context, options);
  };
  async createPlan(plan: CreateRequest_9, options?: CreatePlanOptions) {
    return createPlan(this.#context, plan, options);
  };
  async updatePlan(
    planId: string,
    plan: UpsertRequest_6,
    options?: UpdatePlanOptions,
  ) {
    return updatePlan(this.#context, planId, plan, options);
  };
  async getPlan(planId: string, options?: GetPlanOptions) {
    return getPlan(this.#context, planId, options);
  };
  async deletePlan(planId: string, options?: DeletePlanOptions) {
    return deletePlan(this.#context, planId, options);
  };
  async archivePlan(planId: string, options?: ArchivePlanOptions) {
    return archivePlan(this.#context, planId, options);
  };
  async publishPlan(planId: string, options?: PublishPlanOptions) {
    return publishPlan(this.#context, planId, options);
  }
}
export class LlmCostOverridesEndpointsClient {
  #context: LlmCostOverridesEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: LlmCostOverridesEndpointsClientOptions,
  ) {
    this.#context = createLlmCostOverridesEndpointsClientContext(
      endpoint,
      options
    );

  }
  listOverrides(options?: ListOverridesOptions) {
    return listOverrides(this.#context, options);
  };
  async createOverride(body: OverrideCreate, options?: CreateOverrideOptions) {
    return createOverride(this.#context, body, options);
  };
  async deleteOverride(priceId: string, options?: DeleteOverrideOptions) {
    return deleteOverride(this.#context, priceId, options);
  }
}
export class LlmCostPricesEndpointsClient {
  #context: LlmCostPricesEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: LlmCostPricesEndpointsClientOptions,
  ) {
    this.#context = createLlmCostPricesEndpointsClientContext(
      endpoint,
      options
    );

  }
  listPrices(options?: ListPricesOptions) {
    return listPrices(this.#context, options);
  };
  async getPrice(priceId: string, options?: GetPriceOptions) {
    return getPrice(this.#context, priceId, options);
  }
}
export class FeatureCostEndpointsClient {
  #context: FeatureCostEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: FeatureCostEndpointsClientOptions,
  ) {
    this.#context = createFeatureCostEndpointsClientContext(endpoint, options);

  }
  async queryCost(featureId: string, options?: QueryCostOptions) {
    return queryCost(this.#context, featureId, options);
  }
}
export class FeaturesEndpointsClient {
  #context: FeaturesEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: FeaturesEndpointsClientOptions,
  ) {
    this.#context = createFeaturesEndpointsClientContext(endpoint, options);

  }
  list(options?: ListOptions_14) {
    return list_14(this.#context, options);
  };
  async create(feature: CreateRequest_8, options?: CreateOptions_9) {
    return create_9(this.#context, feature, options);
  };
  async get(featureId: string, options?: GetOptions_10) {
    return get_10(this.#context, featureId, options);
  };
  async update(
    featureId: string,
    feature: FeatureUpdateRequest,
    options?: UpdateOptions_3,
  ) {
    return update_3(this.#context, featureId, feature, options);
  };
  async delete_(featureId: string, options?: DeleteOptions_5) {
    return delete__5(this.#context, featureId, options);
  }
}
export class CurrenciesCustomCostBasesEndpointsClient {
  #context: CurrenciesCustomCostBasesEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: CurrenciesCustomCostBasesEndpointsClientOptions,
  ) {
    this.#context = createCurrenciesCustomCostBasesEndpointsClientContext(
      endpoint,
      options
    );

  }
  getCostBases(currencyId: string, options?: GetCostBasesOptions) {
    return getCostBases(this.#context, currencyId, options);
  };
  async createCostBasis(
    currencyId: string,
    body: CreateRequest_7,
    options?: CreateCostBasisOptions,
  ) {
    return createCostBasis(this.#context, currencyId, body, options);
  }
}
export class CurrenciesCustomEndpointsClient {
  #context: CurrenciesCustomEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: CurrenciesCustomEndpointsClientOptions,
  ) {
    this.#context = createCurrenciesCustomEndpointsClientContext(
      endpoint,
      options
    );

  }
  async create(body: CreateRequest_6, options?: CreateOptions_8) {
    return create_8(this.#context, body, options);
  }
}
export class CurrenciesEndpointsClient {
  #context: CurrenciesEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: CurrenciesEndpointsClientOptions,
  ) {
    this.#context = createCurrenciesEndpointsClientContext(endpoint, options);

  }
  list(options?: ListOptions_13) {
    return list_13(this.#context, options);
  }
}
export class TaxCodesEndpointsClient {
  #context: TaxCodesEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: TaxCodesEndpointsClientOptions,
  ) {
    this.#context = createTaxCodesEndpointsClientContext(endpoint, options);

  }
  async create(taxCode: CreateRequest_5, options?: CreateOptions_7) {
    return create_7(this.#context, taxCode, options);
  };
  async get(taxCodeId: string, options?: GetOptions_9) {
    return get_9(this.#context, taxCodeId, options);
  };
  list(options?: ListOptions_12) {
    return list_12(this.#context, options);
  };
  async upsert(
    taxCodeId: string,
    taxCode: UpsertRequest_5,
    options?: UpsertOptions_3,
  ) {
    return upsert_3(this.#context, taxCodeId, taxCode, options);
  };
  async delete_(taxCodeId: string, options?: DeleteOptions_4) {
    return delete__4(this.#context, taxCodeId, options);
  }
}
export class BillingProfilesEndpointsClient {
  #context: BillingProfilesEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: BillingProfilesEndpointsClientOptions,
  ) {
    this.#context = createBillingProfilesEndpointsClientContext(
      endpoint,
      options
    );

  }
  list(options?: ListOptions_11) {
    return list_11(this.#context, options);
  };
  async create(profile: CreateRequest_4, options?: CreateOptions_6) {
    return create_6(this.#context, profile, options);
  };
  async get(id: string, options?: GetOptions_8) {
    return get_8(this.#context, id, options);
  };
  async update(
    id: string,
    profile: UpsertRequest_4,
    options?: UpdateOptions_2,
  ) {
    return update_2(this.#context, id, profile, options);
  };
  async delete_(id: string, options?: DeleteOptions_3) {
    return delete__3(this.#context, id, options);
  }
}
export class AppsEndpointsClient {
  #context: AppsEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: AppsEndpointsClientOptions,
  ) {
    this.#context = createAppsEndpointsClientContext(endpoint, options);

  }
  list(options?: ListOptions_10) {
    return list_10(this.#context, options);
  };
  async get(appId: string, options?: GetOptions_7) {
    return get_7(this.#context, appId, options);
  }
}
export class SubscriptionAddonEndpointsClient {
  #context: SubscriptionAddonEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: SubscriptionAddonEndpointsClientOptions,
  ) {
    this.#context = createSubscriptionAddonEndpointsClientContext(
      endpoint,
      options
    );

  }
  list(subscriptionId: string, options?: ListOptions_9) {
    return list_9(this.#context, subscriptionId, options);
  }
}
export class SubscriptionsEndpointsClient {
  #context: SubscriptionsEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: SubscriptionsEndpointsClientOptions,
  ) {
    this.#context = createSubscriptionsEndpointsClientContext(
      endpoint,
      options
    );

  }
  async create(subscription: SubscriptionCreate, options?: CreateOptions_5) {
    return create_5(this.#context, subscription, options);
  };
  list(options?: ListOptions_8) {
    return list_8(this.#context, options);
  };
  async get(subscriptionId: string, options?: GetOptions_6) {
    return get_6(this.#context, subscriptionId, options);
  };
  async cancel(
    subscriptionId: string,
    body: SubscriptionCancel,
    options?: CancelOptions,
  ) {
    return cancel(this.#context, subscriptionId, body, options);
  };
  async unscheduleCancelation(
    subscriptionId: string,
    options?: UnscheduleCancelationOptions,
  ) {
    return unscheduleCancelation(this.#context, subscriptionId, options);
  };
  async change(
    subscriptionId: string,
    body: SubscriptionChange,
    options?: ChangeOptions,
  ) {
    return change(this.#context, subscriptionId, body, options);
  }
}
export class CustomerChargesEndpointsClient {
  #context: CustomerChargesEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: CustomerChargesEndpointsClientOptions,
  ) {
    this.#context = createCustomerChargesEndpointsClientContext(
      endpoint,
      options
    );

  }
  list(customerId: string, options?: ListOptions_7) {
    return list_7(this.#context, customerId, options);
  }
}
export class CustomerCreditTransactionEndpointsClient {
  #context: CustomerCreditTransactionEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: CustomerCreditTransactionEndpointsClientOptions,
  ) {
    this.#context = createCustomerCreditTransactionEndpointsClientContext(
      endpoint,
      options
    );

  }
  list(customerId: string, options?: ListOptions_6) {
    return list_6(this.#context, customerId, options);
  }
}
export class CustomerCreditGrantEndpointsClient {
  #context: CustomerCreditGrantEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: CustomerCreditGrantEndpointsClientOptions,
  ) {
    this.#context = createCustomerCreditGrantEndpointsClientContext(
      endpoint,
      options
    );

  }
  async updateExternalSettlement(
    customerId: string,
    creditGrantId: string,
    body: UpdateCreditGrantExternalSettlementRequest,
    options?: UpdateExternalSettlementOptions,
  ) {
    return updateExternalSettlement(
      this.#context,
      customerId,
      creditGrantId,
      body,
      options
    );
  }
}
export class CustomerCreditAdjustmentsEndpointsClient {
  #context: CustomerCreditAdjustmentsEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: CustomerCreditAdjustmentsEndpointsClientOptions,
  ) {
    this.#context = createCustomerCreditAdjustmentsEndpointsClientContext(
      endpoint,
      options
    );

  }
  async create(
    customerId: string,
    creditAdjustment: CreateRequest_3,
    options?: CreateOptions_4,
  ) {
    return create_4(this.#context, customerId, creditAdjustment, options);
  }
}
export class CustomerCreditBalanceEndpointsClient {
  #context: CustomerCreditBalanceEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: CustomerCreditBalanceEndpointsClientOptions,
  ) {
    this.#context = createCustomerCreditBalanceEndpointsClientContext(
      endpoint,
      options
    );

  }
  async get(customerId: string, options?: GetOptions_5) {
    return get_5(this.#context, customerId, options);
  }
}
export class CustomerCreditGrantsEndpointsClient {
  #context: CustomerCreditGrantsEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: CustomerCreditGrantsEndpointsClientOptions,
  ) {
    this.#context = createCustomerCreditGrantsEndpointsClientContext(
      endpoint,
      options
    );

  }
  async create(
    customerId: string,
    creditGrant: CreateRequestNested,
    options?: CreateOptions_3,
  ) {
    return create_3(this.#context, customerId, creditGrant, options);
  };
  async get(customerId: string, creditGrantId: string, options?: GetOptions_4) {
    return get_4(this.#context, customerId, creditGrantId, options);
  };
  list(customerId: string, options?: ListOptions_5) {
    return list_5(this.#context, customerId, options);
  }
}
export class CustomerEntitlementsEndpointsClient {
  #context: CustomerEntitlementsEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: CustomerEntitlementsEndpointsClientOptions,
  ) {
    this.#context = createCustomerEntitlementsEndpointsClientContext(
      endpoint,
      options
    );

  }
  async list(customerId: string, options?: ListOptions_4) {
    return list_4(this.#context, customerId, options);
  }
}
export class CustomerBillingEndpointsClient {
  #context: CustomerBillingEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: CustomerBillingEndpointsClientOptions,
  ) {
    this.#context = createCustomerBillingEndpointsClientContext(
      endpoint,
      options
    );

  }
  async get(customerId: string, options?: GetOptions_3) {
    return get_3(this.#context, customerId, options);
  };
  async upsert(
    customerId: string,
    body: UpsertRequest_2,
    options?: UpsertOptions_2,
  ) {
    return upsert_2(this.#context, customerId, body, options);
  };
  async upsertAppData(
    customerId: string,
    body: UpsertRequest_3,
    options?: UpsertAppDataOptions,
  ) {
    return upsertAppData(this.#context, customerId, body, options);
  };
  async createCheckoutSession(
    customerId: string,
    body: CustomerBillingStripeCreateCheckoutSessionRequest,
    options?: CreateCheckoutSessionOptions,
  ) {
    return createCheckoutSession(this.#context, customerId, body, options);
  };
  async createPortalSession(
    customerId: string,
    body: CustomerBillingStripeCreateCustomerPortalSessionRequest,
    options?: CreatePortalSessionOptions,
  ) {
    return createPortalSession(this.#context, customerId, body, options);
  }
}
export class CustomersEndpointsClient {
  #context: CustomersEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: CustomersEndpointsClientOptions,
  ) {
    this.#context = createCustomersEndpointsClientContext(endpoint, options);

  }
  async create(customer: CreateRequest_2, options?: CreateOptions_2) {
    return create_2(this.#context, customer, options);
  };
  async get(customerId: string, options?: GetOptions_2) {
    return get_2(this.#context, customerId, options);
  };
  list(options?: ListOptions_3) {
    return list_3(this.#context, options);
  };
  async upsert(
    customerId: string,
    customer: UpsertRequest,
    options?: UpsertOptions,
  ) {
    return upsert(this.#context, customerId, customer, options);
  };
  async delete_(customerId: string, options?: DeleteOptions_2) {
    return delete__2(this.#context, customerId, options);
  }
}
export class MetersQueryEndpointsClient {
  #context: MetersQueryEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: MetersQueryEndpointsClientOptions,
  ) {
    this.#context = createMetersQueryEndpointsClientContext(endpoint, options);

  }
  async query(
    meterId: string,
    request: MeterQueryRequest,
    options?: QueryOptions,
  ) {
    return query(this.#context, meterId, request, options);
  };
  async queryCsv(meterId: string, options?: QueryCsvOptions) {
    return queryCsv(this.#context, meterId, options);
  }
}
export class MetersEndpointsClient {
  #context: MetersEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: MetersEndpointsClientOptions,
  ) {
    this.#context = createMetersEndpointsClientContext(endpoint, options);

  }
  async create(meter: CreateRequest, options?: CreateOptions) {
    return create(this.#context, meter, options);
  };
  async get(meterId: string, options?: GetOptions) {
    return get(this.#context, meterId, options);
  };
  list(options?: ListOptions_2) {
    return list_2(this.#context, options);
  };
  async update(meterId: string, meter: UpdateRequest, options?: UpdateOptions) {
    return update(this.#context, meterId, meter, options);
  };
  async delete_(meterId: string, options?: DeleteOptions) {
    return delete_(this.#context, meterId, options);
  }
}
export class EventsEndpointsClient {
  #context: EventsEndpointsClientContext
  constructor(
    endpoint: "http://localhost:{port}/api/v3" | "https://openmeter.cloud/api/v3" | string,
    options?: EventsEndpointsClientOptions,
  ) {
    this.#context = createEventsEndpointsClientContext(endpoint, options);

  }
  list(options?: ListOptions) {
    return list(this.#context, options);
  };
  async ingestEvent(body: MeteringEvent, options?: IngestEventOptions) {
    return ingestEvent(this.#context, body, options);
  };
  async ingestEvents(
    body: Array<MeteringEvent>,
    options?: IngestEventsOptions,
  ) {
    return ingestEvents(this.#context, body, options);
  };
  async ingestEventsJson(
    body: MeteringEvent | Array<MeteringEvent>,
    options?: IngestEventsJsonOptions,
  ) {
    return ingestEventsJson(this.#context, body, options);
  }
}
