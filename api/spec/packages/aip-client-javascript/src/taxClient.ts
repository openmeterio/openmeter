import {
  createTaxClientContext,
  type TaxClientContext,
  type TaxClientOptions,
} from "./api/taxClientContext.js";
import {
  createTaxCodesOperationsClientContext,
  type TaxCodesOperationsClientContext,
  type TaxCodesOperationsClientOptions,
} from "./api/taxCodesOperationsClient/taxCodesOperationsClientContext.js";
import {
  create,
  type CreateOptions,
  delete_,
  type DeleteOptions,
  get,
  type GetOptions,
  list,
  type ListOptions,
  upsert,
  type UpsertOptions,
} from "./api/taxCodesOperationsClient/taxCodesOperationsClientOperations.js";
import type { CreateRequest_5, UpsertRequest_5 } from "./models/models.js";

export class TaxClient {
  #context: TaxClientContext
  taxCodesOperationsClient: TaxCodesOperationsClient
  constructor(endpoint: string, options?: TaxClientOptions) {
    this.#context = createTaxClientContext(endpoint, options);
    this.taxCodesOperationsClient = new TaxCodesOperationsClient(
      endpoint,
      options
    );
  }
}
export class TaxCodesOperationsClient {
  #context: TaxCodesOperationsClientContext
  constructor(endpoint: string, options?: TaxCodesOperationsClientOptions) {
    this.#context = createTaxCodesOperationsClientContext(endpoint, options);

  }
  async create(taxCode: CreateRequest_5, options?: CreateOptions) {
    return create(this.#context, taxCode, options);
  };
  async get(taxCodeId: string, options?: GetOptions) {
    return get(this.#context, taxCodeId, options);
  };
  list(options?: ListOptions) {
    return list(this.#context, options);
  };
  async upsert(
    taxCodeId: string,
    taxCode: UpsertRequest_5,
    options?: UpsertOptions,
  ) {
    return upsert(this.#context, taxCodeId, taxCode, options);
  };
  async delete_(taxCodeId: string, options?: DeleteOptions) {
    return delete_(this.#context, taxCodeId, options);
  }
}
