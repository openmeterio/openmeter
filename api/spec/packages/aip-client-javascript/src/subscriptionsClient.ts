import {
  createSubscriptionAddonOperationsClientContext,
  type SubscriptionAddonOperationsClientContext,
  type SubscriptionAddonOperationsClientOptions,
} from "./api/subscriptionAddonOperationsClient/subscriptionAddonOperationsClientContext.js";
import {
  list as list_2,
  type ListOptions as ListOptions_2,
} from "./api/subscriptionAddonOperationsClient/subscriptionAddonOperationsClientOperations.js";
import {
  createSubscriptionsClientContext,
  type SubscriptionsClientContext,
  type SubscriptionsClientOptions,
} from "./api/subscriptionsClientContext.js";
import {
  createSubscriptionsOperationsClientContext,
  type SubscriptionsOperationsClientContext,
  type SubscriptionsOperationsClientOptions,
} from "./api/subscriptionsOperationsClient/subscriptionsOperationsClientContext.js";
import {
  cancel,
  type CancelOptions,
  change,
  type ChangeOptions,
  create,
  type CreateOptions,
  get,
  type GetOptions,
  list,
  type ListOptions,
  unscheduleCancelation,
  type UnscheduleCancelationOptions,
} from "./api/subscriptionsOperationsClient/subscriptionsOperationsClientOperations.js";
import type {
  SubscriptionCancel,
  SubscriptionChange,
  SubscriptionCreate,
} from "./models/models.js";

export class SubscriptionsClient {
  #context: SubscriptionsClientContext
  subscriptionsOperationsClient: SubscriptionsOperationsClient;
  subscriptionAddonOperationsClient: SubscriptionAddonOperationsClient
  constructor(endpoint: string, options?: SubscriptionsClientOptions) {
    this.#context = createSubscriptionsClientContext(endpoint, options);
    this.subscriptionsOperationsClient = new SubscriptionsOperationsClient(
      endpoint,
      options
    );;this
      .subscriptionAddonOperationsClient = new SubscriptionAddonOperationsClient(
      endpoint,
      options
    );
  }
}
export class SubscriptionAddonOperationsClient {
  #context: SubscriptionAddonOperationsClientContext
  constructor(
    endpoint: string,
    options?: SubscriptionAddonOperationsClientOptions,
  ) {
    this.#context = createSubscriptionAddonOperationsClientContext(
      endpoint,
      options
    );

  }
  list(subscriptionId: string, options?: ListOptions_2) {
    return list_2(this.#context, subscriptionId, options);
  }
}
export class SubscriptionsOperationsClient {
  #context: SubscriptionsOperationsClientContext
  constructor(
    endpoint: string,
    options?: SubscriptionsOperationsClientOptions,
  ) {
    this.#context = createSubscriptionsOperationsClientContext(
      endpoint,
      options
    );

  }
  async create(subscription: SubscriptionCreate, options?: CreateOptions) {
    return create(this.#context, subscription, options);
  };
  list(options?: ListOptions) {
    return list(this.#context, options);
  };
  async get(subscriptionId: string, options?: GetOptions) {
    return get(this.#context, subscriptionId, options);
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
