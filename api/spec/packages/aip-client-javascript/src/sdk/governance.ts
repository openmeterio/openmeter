import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  queryGovernanceAccess,
} from '../funcs/governance.js'
import type {
  QueryGovernanceAccessRequest,
  QueryGovernanceAccessResponse,
} from '../models/operations/governance.js'

export class Governance {
  constructor(private readonly _client: Client) {}

  async queryAccess(
    request: QueryGovernanceAccessRequest,
    options?: RequestOptions,
  ): Promise<QueryGovernanceAccessResponse> {
    return unwrap(await queryGovernanceAccess(this._client, request, options))
  }
}
