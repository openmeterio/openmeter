import {
  createDefaultsClientContext,
  type DefaultsClientContext,
  type DefaultsClientOptions,
} from "./api/defaultsClientContext.js";
import {
  createOrganizationDefaultTaxCodesOperationsClientContext,
  type OrganizationDefaultTaxCodesOperationsClientContext,
  type OrganizationDefaultTaxCodesOperationsClientOptions,
} from "./api/organizationDefaultTaxCodesOperationsClient/organizationDefaultTaxCodesOperationsClientContext.js";
import {
  get,
  type GetOptions,
  update,
  type UpdateOptions,
} from "./api/organizationDefaultTaxCodesOperationsClient/organizationDefaultTaxCodesOperationsClientOperations.js";
import type { UpdateRequest_2 } from "./models/models.js";

export class DefaultsClient {
  #context: DefaultsClientContext
  organizationDefaultTaxCodesOperationsClient: OrganizationDefaultTaxCodesOperationsClient
  constructor(endpoint: string, options?: DefaultsClientOptions) {
    this.#context = createDefaultsClientContext(endpoint, options);
    this
      .organizationDefaultTaxCodesOperationsClient = new OrganizationDefaultTaxCodesOperationsClient(
      endpoint,
      options
    );
  }
}
export class OrganizationDefaultTaxCodesOperationsClient {
  #context: OrganizationDefaultTaxCodesOperationsClientContext
  constructor(
    endpoint: string,
    options?: OrganizationDefaultTaxCodesOperationsClientOptions,
  ) {
    this.#context = createOrganizationDefaultTaxCodesOperationsClientContext(
      endpoint,
      options
    );

  }
  async get(options?: GetOptions) {
    return get(this.#context, options);
  };
  async update(body: UpdateRequest_2, options?: UpdateOptions) {
    return update(this.#context, body, options);
  }
}
