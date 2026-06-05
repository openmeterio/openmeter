import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
import type {
  QueryGovernanceAccessRequest,
  QueryGovernanceAccessResponse,
} from '../models/operations/governance.js'

export function queryGovernanceAccess(
  client: Client,
  req: QueryGovernanceAccessRequest,
  options?: RequestOptions,
): Promise<Result<QueryGovernanceAccessResponse>> {
  const searchParams = toURLSearchParams({
    page: req.page,
  })
  return request(() =>
    http(client)
      .post('openmeter/governance/query', { ...options, searchParams, json: req.body })
      .json<QueryGovernanceAccessResponse>(),
  )
}
