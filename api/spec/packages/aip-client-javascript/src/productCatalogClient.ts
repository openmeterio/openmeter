import {
  type AddonOperationsClientContext,
  type AddonOperationsClientOptions,
  createAddonOperationsClientContext,
} from "./api/addonOperationsClient/addonOperationsClientContext.js";
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
} from "./api/addonOperationsClient/addonOperationsClientOperations.js";
import {
  createPlanAddonOperationsClientContext,
  type PlanAddonOperationsClientContext,
  type PlanAddonOperationsClientOptions,
} from "./api/planAddonOperationsClient/planAddonOperationsClientContext.js";
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
} from "./api/planAddonOperationsClient/planAddonOperationsClientOperations.js";
import {
  createPlanOperationsClientContext,
  type PlanOperationsClientContext,
  type PlanOperationsClientOptions,
} from "./api/planOperationsClient/planOperationsClientContext.js";
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
} from "./api/planOperationsClient/planOperationsClientOperations.js";
import {
  createProductCatalogClientContext,
  type ProductCatalogClientContext,
  type ProductCatalogClientOptions,
} from "./api/productCatalogClientContext.js";
import type {
  CreateRequest_10,
  CreateRequest_11,
  CreateRequest_9,
  UpsertRequest_6,
  UpsertRequest_7,
  UpsertRequest_8,
} from "./models/models.js";

export class ProductCatalogClient {
  #context: ProductCatalogClientContext
  planOperationsClient: PlanOperationsClient;
  addonOperationsClient: AddonOperationsClient;
  planAddonOperationsClient: PlanAddonOperationsClient
  constructor(endpoint: string, options?: ProductCatalogClientOptions) {
    this.#context = createProductCatalogClientContext(endpoint, options);
    this.planOperationsClient = new PlanOperationsClient(
      endpoint,
      options
    );;this.addonOperationsClient = new AddonOperationsClient(
      endpoint,
      options
    );;this.planAddonOperationsClient = new PlanAddonOperationsClient(
      endpoint,
      options
    );
  }
}
export class PlanAddonOperationsClient {
  #context: PlanAddonOperationsClientContext
  constructor(endpoint: string, options?: PlanAddonOperationsClientOptions) {
    this.#context = createPlanAddonOperationsClientContext(endpoint, options);

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
export class AddonOperationsClient {
  #context: AddonOperationsClientContext
  constructor(endpoint: string, options?: AddonOperationsClientOptions) {
    this.#context = createAddonOperationsClientContext(endpoint, options);

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
export class PlanOperationsClient {
  #context: PlanOperationsClientContext
  constructor(endpoint: string, options?: PlanOperationsClientOptions) {
    this.#context = createPlanOperationsClientContext(endpoint, options);

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
