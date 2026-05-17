import {
  createFeatureCostOperationsClientContext,
  type FeatureCostOperationsClientContext,
  type FeatureCostOperationsClientOptions,
} from "./api/featureCostOperationsClient/featureCostOperationsClientContext.js";
import {
  queryCost,
  type QueryCostOptions,
} from "./api/featureCostOperationsClient/featureCostOperationsClientOperations.js";
import {
  createFeatureOperationsClientContext,
  type FeatureOperationsClientContext,
  type FeatureOperationsClientOptions,
} from "./api/featureOperationsClient/featureOperationsClientContext.js";
import {
  create,
  type CreateOptions,
  delete_,
  type DeleteOptions,
  get,
  type GetOptions,
  list,
  type ListOptions,
  update,
  type UpdateOptions,
} from "./api/featureOperationsClient/featureOperationsClientOperations.js";
import {
  createFeaturesClientContext,
  type FeaturesClientContext,
  type FeaturesClientOptions,
} from "./api/featuresClientContext.js";
import type { CreateRequest_8, FeatureUpdateRequest } from "./models/models.js";

export class FeaturesClient {
  #context: FeaturesClientContext
  featureOperationsClient: FeatureOperationsClient;
  featureCostOperationsClient: FeatureCostOperationsClient
  constructor(endpoint: string, options?: FeaturesClientOptions) {
    this.#context = createFeaturesClientContext(endpoint, options);
    this.featureOperationsClient = new FeatureOperationsClient(
      endpoint,
      options
    );;this.featureCostOperationsClient = new FeatureCostOperationsClient(
      endpoint,
      options
    );
  }
}
export class FeatureCostOperationsClient {
  #context: FeatureCostOperationsClientContext
  constructor(endpoint: string, options?: FeatureCostOperationsClientOptions) {
    this.#context = createFeatureCostOperationsClientContext(endpoint, options);

  }
  async queryCost(featureId: string, options?: QueryCostOptions) {
    return queryCost(this.#context, featureId, options);
  }
}
export class FeatureOperationsClient {
  #context: FeatureOperationsClientContext
  constructor(endpoint: string, options?: FeatureOperationsClientOptions) {
    this.#context = createFeatureOperationsClientContext(endpoint, options);

  }
  list(options?: ListOptions) {
    return list(this.#context, options);
  };
  async create(feature: CreateRequest_8, options?: CreateOptions) {
    return create(this.#context, feature, options);
  };
  async get(featureId: string, options?: GetOptions) {
    return get(this.#context, featureId, options);
  };
  async update(
    featureId: string,
    feature: FeatureUpdateRequest,
    options?: UpdateOptions,
  ) {
    return update(this.#context, featureId, feature, options);
  };
  async delete_(featureId: string, options?: DeleteOptions) {
    return delete_(this.#context, featureId, options);
  }
}
