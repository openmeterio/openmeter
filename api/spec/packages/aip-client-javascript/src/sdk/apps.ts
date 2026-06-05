import { type Client } from '../core.js'
import { unwrap, type RequestOptions } from '../lib/types.js'
import {
  listApps,
  getApp,
} from '../funcs/apps.js'
import type {
  ListAppsRequest,
  ListAppsResponse,
  GetAppRequest,
  GetAppResponse,
} from '../models/operations/apps.js'

export class Apps {
  constructor(private readonly _client: Client) {}

  async list(
    request?: ListAppsRequest,
    options?: RequestOptions,
  ): Promise<ListAppsResponse> {
    return unwrap(await listApps(this._client, request, options))
  }

  async get(
    request: GetAppRequest,
    options?: RequestOptions,
  ): Promise<GetAppResponse> {
    return unwrap(await getApp(this._client, request, options))
  }
}
