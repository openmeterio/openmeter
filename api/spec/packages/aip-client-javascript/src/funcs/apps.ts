import { type Client, http } from '../core.js'
import { type Result, type RequestOptions } from '../lib/types.js'
import { request } from '../lib/request.js'
import { encodePath, toURLSearchParams, encodeSort } from '../lib/encodings.js'
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
  const searchParams = toURLSearchParams({
    page: req.page,
  })
  return request(() =>
    http(client)
      .get('openmeter/apps', { ...options, searchParams })
      .json<ListAppsResponse>(),
  )
}

export function getApp(
  client: Client,
  req: GetAppRequest,
  options?: RequestOptions,
): Promise<Result<GetAppResponse>> {
  const path = encodePath('openmeter/apps/{appId}', { appId: req.appId })
  return request(() => http(client).get(path, options).json<GetAppResponse>())
}
