import ky, { type KyInstance as HTTPClient } from 'ky'
import { type SDKOptions } from './lib/config.js'
import { encodePath } from './lib/encodings.js'

export class Client {
  readonly _options: SDKOptions
  readonly _http: HTTPClient

  constructor(options: SDKOptions) {
    this._options = options

    let baseUrl =
      typeof options.baseUrl === 'string'
        ? encodePath(options.baseUrl, options.serverVariables ?? {})
        : String(options.baseUrl)
    if (!baseUrl.endsWith('/')) {
      baseUrl = `${baseUrl}/`
    }

    this._http = ky.create({
      ...options,
      baseUrl,
      prefix: undefined,
      hooks: {
        ...options.hooks,
        beforeRequest: [
          ...(options.hooks?.beforeRequest ?? []),
          async ({ request }) => {
            const token =
              typeof options.apiKey === 'function'
                ? await options.apiKey()
                : options.apiKey
            if (token) {
              request.headers.set('Authorization', `Bearer ${token}`)
            }
          },
        ],
      },
    })
  }
}

export function http(client: Client): HTTPClient {
  return client._http
}

export type { HTTPClient }
