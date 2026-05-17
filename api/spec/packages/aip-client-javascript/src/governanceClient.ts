import {
  createGovernanceClientContext,
  type GovernanceClientContext,
  type GovernanceClientOptions,
} from "./api/governanceClientContext.js";
import {
  createGovernanceOperationsClientContext,
  type GovernanceOperationsClientContext,
  type GovernanceOperationsClientOptions,
} from "./api/governanceOperationsClient/governanceOperationsClientContext.js";
import {
  query,
  type QueryOptions,
} from "./api/governanceOperationsClient/governanceOperationsClientOperations.js";
import type { GovernanceQueryRequest } from "./models/models.js";

export class GovernanceClient {
  #context: GovernanceClientContext
  governanceOperationsClient: GovernanceOperationsClient
  constructor(endpoint: string, options?: GovernanceClientOptions) {
    this.#context = createGovernanceClientContext(endpoint, options);
    this.governanceOperationsClient = new GovernanceOperationsClient(
      endpoint,
      options
    );
  }
}
export class GovernanceOperationsClient {
  #context: GovernanceOperationsClientContext
  constructor(endpoint: string, options?: GovernanceOperationsClientOptions) {
    this.#context = createGovernanceOperationsClientContext(endpoint, options);

  }
  async query(_: GovernanceQueryRequest, options?: QueryOptions) {
    return query(this.#context, _, options);
  }
}
