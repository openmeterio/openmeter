import {
  createCurrenciesClientContext,
  type CurrenciesClientContext,
  type CurrenciesClientOptions,
} from "./api/currenciesClientContext.js";
import {
  createCurrenciesCustomCostBasesOperationsClientContext,
  type CurrenciesCustomCostBasesOperationsClientContext,
  type CurrenciesCustomCostBasesOperationsClientOptions,
} from "./api/currenciesCustomCostBasesOperationsClient/currenciesCustomCostBasesOperationsClientContext.js";
import {
  createCostBasis,
  type CreateCostBasisOptions,
  getCostBases,
  type GetCostBasesOptions,
} from "./api/currenciesCustomCostBasesOperationsClient/currenciesCustomCostBasesOperationsClientOperations.js";
import {
  createCurrenciesCustomOperationsClientContext,
  type CurrenciesCustomOperationsClientContext,
  type CurrenciesCustomOperationsClientOptions,
} from "./api/currenciesCustomOperationsClient/currenciesCustomOperationsClientContext.js";
import {
  create,
  type CreateOptions,
} from "./api/currenciesCustomOperationsClient/currenciesCustomOperationsClientOperations.js";
import {
  createCurrenciesOperationsClientContext,
  type CurrenciesOperationsClientContext,
  type CurrenciesOperationsClientOptions,
} from "./api/currenciesOperationsClient/currenciesOperationsClientContext.js";
import {
  list,
  type ListOptions,
} from "./api/currenciesOperationsClient/currenciesOperationsClientOperations.js";
import type { CreateRequest_6, CreateRequest_7 } from "./models/models.js";

export class CurrenciesClient {
  #context: CurrenciesClientContext
  currenciesOperationsClient: CurrenciesOperationsClient;
  currenciesCustomOperationsClient: CurrenciesCustomOperationsClient;
  currenciesCustomCostBasesOperationsClient: CurrenciesCustomCostBasesOperationsClient
  constructor(endpoint: string, options?: CurrenciesClientOptions) {
    this.#context = createCurrenciesClientContext(endpoint, options);
    this.currenciesOperationsClient = new CurrenciesOperationsClient(
      endpoint,
      options
    );;this
      .currenciesCustomOperationsClient = new CurrenciesCustomOperationsClient(
      endpoint,
      options
    );;this
      .currenciesCustomCostBasesOperationsClient = new CurrenciesCustomCostBasesOperationsClient(
      endpoint,
      options
    );
  }
}
export class CurrenciesCustomCostBasesOperationsClient {
  #context: CurrenciesCustomCostBasesOperationsClientContext
  constructor(
    endpoint: string,
    options?: CurrenciesCustomCostBasesOperationsClientOptions,
  ) {
    this.#context = createCurrenciesCustomCostBasesOperationsClientContext(
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
export class CurrenciesCustomOperationsClient {
  #context: CurrenciesCustomOperationsClientContext
  constructor(
    endpoint: string,
    options?: CurrenciesCustomOperationsClientOptions,
  ) {
    this.#context = createCurrenciesCustomOperationsClientContext(
      endpoint,
      options
    );

  }
  async create(body: CreateRequest_6, options?: CreateOptions) {
    return create(this.#context, body, options);
  }
}
export class CurrenciesOperationsClient {
  #context: CurrenciesOperationsClientContext
  constructor(endpoint: string, options?: CurrenciesOperationsClientOptions) {
    this.#context = createCurrenciesOperationsClientContext(endpoint, options);

  }
  list(options?: ListOptions) {
    return list(this.#context, options);
  }
}
