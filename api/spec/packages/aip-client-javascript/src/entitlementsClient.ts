import {
  createCustomerEntitlementsOperationsClientContext,
  type CustomerEntitlementsOperationsClientContext,
  type CustomerEntitlementsOperationsClientOptions,
} from "./api/customerEntitlementsOperationsClient/customerEntitlementsOperationsClientContext.js";
import {
  list,
  type ListOptions,
} from "./api/customerEntitlementsOperationsClient/customerEntitlementsOperationsClientOperations.js";
import {
  createEntitlementsClientContext,
  type EntitlementsClientContext,
  type EntitlementsClientOptions,
} from "./api/entitlementsClientContext.js";

export class EntitlementsClient {
  #context: EntitlementsClientContext
  customerEntitlementsOperationsClient: CustomerEntitlementsOperationsClient
  constructor(endpoint: string, options?: EntitlementsClientOptions) {
    this.#context = createEntitlementsClientContext(endpoint, options);
    this
      .customerEntitlementsOperationsClient = new CustomerEntitlementsOperationsClient(
      endpoint,
      options
    );
  }
}
export class CustomerEntitlementsOperationsClient {
  #context: CustomerEntitlementsOperationsClientContext
  constructor(
    endpoint: string,
    options?: CustomerEntitlementsOperationsClientOptions,
  ) {
    this.#context = createCustomerEntitlementsOperationsClientContext(
      endpoint,
      options
    );

  }
  async list(customerId: string, options?: ListOptions) {
    return list(this.#context, customerId, options);
  }
}
