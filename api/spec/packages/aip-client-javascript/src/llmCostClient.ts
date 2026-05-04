import {
  createLlmCostClientContext,
  type LlmCostClientContext,
  type LlmCostClientOptions,
} from "./api/llmCostClientContext.js";
import {
  createLlmCostOverridesOperationsClientContext,
  type LlmCostOverridesOperationsClientContext,
  type LlmCostOverridesOperationsClientOptions,
} from "./api/llmCostOverridesOperationsClient/llmCostOverridesOperationsClientContext.js";
import {
  createOverride,
  type CreateOverrideOptions,
  deleteOverride,
  type DeleteOverrideOptions,
  listOverrides,
  type ListOverridesOptions,
} from "./api/llmCostOverridesOperationsClient/llmCostOverridesOperationsClientOperations.js";
import {
  createLlmCostPricesOperationsClientContext,
  type LlmCostPricesOperationsClientContext,
  type LlmCostPricesOperationsClientOptions,
} from "./api/llmCostPricesOperationsClient/llmCostPricesOperationsClientContext.js";
import {
  getPrice,
  type GetPriceOptions,
  listPrices,
  type ListPricesOptions,
} from "./api/llmCostPricesOperationsClient/llmCostPricesOperationsClientOperations.js";
import type { OverrideCreate } from "./models/models.js";

export class LlmCostClient {
  #context: LlmCostClientContext
  llmCostPricesOperationsClient: LlmCostPricesOperationsClient;
  llmCostOverridesOperationsClient: LlmCostOverridesOperationsClient
  constructor(endpoint: string, options?: LlmCostClientOptions) {
    this.#context = createLlmCostClientContext(endpoint, options);
    this.llmCostPricesOperationsClient = new LlmCostPricesOperationsClient(
      endpoint,
      options
    );;this
      .llmCostOverridesOperationsClient = new LlmCostOverridesOperationsClient(
      endpoint,
      options
    );
  }
}
export class LlmCostOverridesOperationsClient {
  #context: LlmCostOverridesOperationsClientContext
  constructor(
    endpoint: string,
    options?: LlmCostOverridesOperationsClientOptions,
  ) {
    this.#context = createLlmCostOverridesOperationsClientContext(
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
export class LlmCostPricesOperationsClient {
  #context: LlmCostPricesOperationsClientContext
  constructor(
    endpoint: string,
    options?: LlmCostPricesOperationsClientOptions,
  ) {
    this.#context = createLlmCostPricesOperationsClientContext(
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
