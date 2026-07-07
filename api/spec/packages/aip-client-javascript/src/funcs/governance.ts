import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
import type {
  QueryGovernanceAccessRequest,
  QueryGovernanceAccessResponse,
} from '../models/operations/governance.js'

export function queryGovernanceAccess(
  client: Client,
  req: QueryGovernanceAccessRequest,
  options?: RequestOptions,
): Promise<Result<QueryGovernanceAccessResponse>> {
  return request(() => {
    const body = toWire(req.body, schemas.queryGovernanceAccessBody)
    if (client._options.validate) {
      assertValid(schemas.queryGovernanceAccessBodyWire, body)
    }
    const query = toWire(
      {
        page: req.page,
      },
      schemas.queryGovernanceAccessQueryParams,
    )
    if (client._options.validate) {
      assertValid(schemas.queryGovernanceAccessQueryParamsWire, query)
    }
    const searchParams = toURLSearchParams(query)
    return http(client)
      .post('openmeter/governance/query', {
        ...options,
        searchParams,
        json: body,
      })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.queryGovernanceAccessResponseWire, data)
        }
        return fromWire(data, schemas.queryGovernanceAccessResponse)
      })
  })
}
