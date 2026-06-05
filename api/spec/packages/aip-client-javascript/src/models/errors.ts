import { z } from 'zod'
import * as schemas from './schemas.js'

export class HTTPError extends Error {
  constructor(
    message: string,
    public type: string,
    public title: string,
    public status: number,
    public url: string,
    protected __raw?: Record<string, unknown>,
  ) {
    super(message)
  }

  static fromResponse(resp: {
    response: Response
    error?: z.infer<typeof schemas.baseError>
  }) {
    if (
      resp.response.headers
        .get('Content-Type')
        ?.includes('application/problem+json') &&
      resp.error
    ) {
      return new HTTPError(
        resp.error.detail,
        resp.error.type ?? resp.error.title,
        resp.error.title,
        resp.error.status ?? resp.response.status,
        resp.response.url,
        resp.error,
      )
    }

    return new HTTPError(
      `Request failed: ${resp.response.statusText}`,
      resp.response.statusText,
      resp.response.statusText,
      resp.response.status,
      resp.response.url,
    )
  }

  getField(key: string) {
    return this.__raw?.[key]
  }
}
