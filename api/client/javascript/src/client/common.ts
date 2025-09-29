import type { UnexpectedProblemResponse } from './schemas.js'

/**
 * Request options
 */
export type RequestOptions = Pick<RequestInit, 'signal'>

/**
 * An error that occurred during an HTTP request
 */
export class HTTPError extends Error {
  public name = 'HTTPError'
  public client = 'OpenMeter'

  constructor(
    public message: string,
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
    error?: UnexpectedProblemResponse
  }): HTTPError {
    if (
      resp.response.headers.get('Content-Type') ===
        'application/problem+json' &&
      resp.error
    ) {
      return new HTTPError(
        `Request failed (${resp.response.url}) [${resp.response.status}]: ${resp.error.detail}`,
        resp.error.type,
        resp.error.title,
        resp.error.status ?? resp.response.status,
        resp.response.url,
        resp.error,
      )
    }

    return new HTTPError(
      `Request failed (${resp.response.url}) [${resp.response.status}]: ${resp.response.statusText}`,
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

/**
 * Check if an error is an HTTPError
 * @param error - The error to check
 * @returns Whether the error is an HTTPError
 */
export function isHTTPError(error: unknown): error is HTTPError {
  return error instanceof HTTPError
}
