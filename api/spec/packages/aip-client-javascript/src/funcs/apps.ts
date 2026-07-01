import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { toURLSearchParams, encodeSort } from '../lib/encodings.js'
import { toWire, fromWire, assertValid } from '../lib/wire.js'
import * as schemas from '../models/schemas.js'
import type {
  ListAppsRequest,
  ListAppsResponse,
  GetAppRequest,
  GetAppResponse,
} from '../models/operations/apps.js'

export function listApps(
  client: Client,
  req: ListAppsRequest = {},
  options?: RequestOptions,
): Promise<Result<ListAppsResponse>> {
  return request(() => {
    const query = toWire(
      {
        page: req.page,
      },
      schemas.listAppsQueryParams,
    )
    if (client._options.validate) {
      assertValid(schemas.listAppsQueryParamsWire, query)
    }
    const searchParams = toURLSearchParams(query)
    return http(client)
      .get('openmeter/apps', { ...options, searchParams })
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.listAppsResponseWire, data)
        }
        return fromWire(data, schemas.listAppsResponse)
      })
  })
}

export function getApp(
  client: Client,
  req: GetAppRequest,
  options?: RequestOptions,
): Promise<Result<GetAppResponse>> {
  return request(() => {
    const path = `openmeter/apps/${(() => {
      if (req.appId === undefined) {
        throw new Error('missing path parameter: appId')
      }
      return encodeURIComponent(String(req.appId))
    })()}`
    return http(client)
      .get(path, options)
      .json()
      .then((data) => {
        if (client._options.validate) {
          assertValid(schemas.getAppResponseWire, data)
        }
        return fromWire(data, schemas.getAppResponse)
      })
  })
}
