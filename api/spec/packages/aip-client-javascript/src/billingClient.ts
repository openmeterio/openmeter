import {
  type BillingClientContext,
  type BillingClientOptions,
  createBillingClientContext,
} from "./api/billingClientContext.js";
import {
  type BillingProfilesOperationsClientContext,
  type BillingProfilesOperationsClientOptions,
  createBillingProfilesOperationsClientContext,
} from "./api/billingProfilesOperationsClient/billingProfilesOperationsClientContext.js";
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
} from "./api/billingProfilesOperationsClient/billingProfilesOperationsClientOperations.js";
import type { CreateRequest_4, UpsertRequest_4 } from "./models/models.js";

export class BillingClient {
  #context: BillingClientContext
  billingProfilesOperationsClient: BillingProfilesOperationsClient
  constructor(endpoint: string, options?: BillingClientOptions) {
    this.#context = createBillingClientContext(endpoint, options);
    this.billingProfilesOperationsClient = new BillingProfilesOperationsClient(
      endpoint,
      options
    );
  }
}
export class BillingProfilesOperationsClient {
  #context: BillingProfilesOperationsClientContext
  constructor(
    endpoint: string,
    options?: BillingProfilesOperationsClientOptions,
  ) {
    this.#context = createBillingProfilesOperationsClientContext(
      endpoint,
      options
    );

  }
  list(options?: ListOptions) {
    return list(this.#context, options);
  };
  async create(profile: CreateRequest_4, options?: CreateOptions) {
    return create(this.#context, profile, options);
  };
  async get(id: string, options?: GetOptions) {
    return get(this.#context, id, options);
  };
  async update(id: string, profile: UpsertRequest_4, options?: UpdateOptions) {
    return update(this.#context, id, profile, options);
  };
  async delete_(id: string, options?: DeleteOptions) {
    return delete_(this.#context, id, options);
  }
}
