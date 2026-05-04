import {
  createFieldFiltersEndpointsClientContext,
  type FieldFiltersEndpointsClientContext,
  type FieldFiltersEndpointsClientOptions,
} from "./api/fieldFiltersEndpointsClient/fieldFiltersEndpointsClientContext.js";
import {
  getFieldFilters,
  type GetFieldFiltersOptions,
} from "./api/fieldFiltersEndpointsClient/fieldFiltersEndpointsClientOperations.js";
import {
  createTestClientContext,
  type TestClientContext,
  type TestClientOptions,
} from "./api/testClientContext.js";

export class TestClient {
  #context: TestClientContext
  fieldFiltersEndpointsClient: FieldFiltersEndpointsClient
  constructor(endpoint: string, options?: TestClientOptions) {
    this.#context = createTestClientContext(endpoint, options);
    this.fieldFiltersEndpointsClient = new FieldFiltersEndpointsClient(
      endpoint,
      options
    );
  }
}
export class FieldFiltersEndpointsClient {
  #context: FieldFiltersEndpointsClientContext
  constructor(endpoint: string, options?: FieldFiltersEndpointsClientOptions) {
    this.#context = createFieldFiltersEndpointsClientContext(endpoint, options);

  }
  async getFieldFilters(options?: GetFieldFiltersOptions) {
    return getFieldFilters(this.#context, options);
  }
}
