import {
  type AppsClientContext,
  type AppsClientOptions,
  createAppsClientContext,
} from "./api/appsClientContext.js";
import {
  type AppsOperationsClientContext,
  type AppsOperationsClientOptions,
  createAppsOperationsClientContext,
} from "./api/appsOperationsClient/appsOperationsClientContext.js";
import {
  get,
  type GetOptions,
  list,
  type ListOptions,
} from "./api/appsOperationsClient/appsOperationsClientOperations.js";

export class AppsClient {
  #context: AppsClientContext
  appsOperationsClient: AppsOperationsClient
  constructor(endpoint: string, options?: AppsClientOptions) {
    this.#context = createAppsClientContext(endpoint, options);
    this.appsOperationsClient = new AppsOperationsClient(endpoint, options);
  }
}
export class AppsOperationsClient {
  #context: AppsOperationsClientContext
  constructor(endpoint: string, options?: AppsOperationsClientOptions) {
    this.#context = createAppsOperationsClientContext(endpoint, options);

  }
  list(options?: ListOptions) {
    return list(this.#context, options);
  };
  async get(appId: string, options?: GetOptions) {
    return get(this.#context, appId, options);
  }
}
