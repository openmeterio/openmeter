import {
  createMetersClientContext,
  type MetersClientContext,
  type MetersClientOptions,
} from "./api/metersClientContext.js";
import {
  createMetersOperationsClientContext,
  type MetersOperationsClientContext,
  type MetersOperationsClientOptions,
} from "./api/metersOperationsClient/metersOperationsClientContext.js";
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
} from "./api/metersOperationsClient/metersOperationsClientOperations.js";
import {
  createMetersQueryOperationsClientContext,
  type MetersQueryOperationsClientContext,
  type MetersQueryOperationsClientOptions,
} from "./api/metersQueryOperationsClient/metersQueryOperationsClientContext.js";
import {
  query,
  queryCsv,
  type QueryCsvOptions,
  type QueryOptions,
} from "./api/metersQueryOperationsClient/metersQueryOperationsClientOperations.js";
import type {
  CreateRequest,
  MeterQueryRequest,
  UpdateRequest,
} from "./models/models.js";

export class MetersClient {
  #context: MetersClientContext
  metersOperationsClient: MetersOperationsClient;
  metersQueryOperationsClient: MetersQueryOperationsClient
  constructor(endpoint: string, options?: MetersClientOptions) {
    this.#context = createMetersClientContext(endpoint, options);
    this.metersOperationsClient = new MetersOperationsClient(
      endpoint,
      options
    );;this.metersQueryOperationsClient = new MetersQueryOperationsClient(
      endpoint,
      options
    );
  }
}
export class MetersQueryOperationsClient {
  #context: MetersQueryOperationsClientContext
  constructor(endpoint: string, options?: MetersQueryOperationsClientOptions) {
    this.#context = createMetersQueryOperationsClientContext(endpoint, options);

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
export class MetersOperationsClient {
  #context: MetersOperationsClientContext
  constructor(endpoint: string, options?: MetersOperationsClientOptions) {
    this.#context = createMetersOperationsClientContext(endpoint, options);

  }
  async create(meter: CreateRequest, options?: CreateOptions) {
    return create(this.#context, meter, options);
  };
  async get(meterId: string, options?: GetOptions) {
    return get(this.#context, meterId, options);
  };
  list(options?: ListOptions) {
    return list(this.#context, options);
  };
  async update(meterId: string, meter: UpdateRequest, options?: UpdateOptions) {
    return update(this.#context, meterId, meter, options);
  };
  async delete_(meterId: string, options?: DeleteOptions) {
    return delete_(this.#context, meterId, options);
  }
}
