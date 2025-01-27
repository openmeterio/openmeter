import type { UnexpectedProblemResponse } from './schemas.js'
import type { FetchResponse, ParseAsResponse } from 'openapi-fetch'
import type {
  MediaType,
  ResponseObjectMap,
  SuccessResponse,
} from 'openapi-typescript-helpers'

// Add more options as needed: 'headers' | 'credentials' | 'mode' | 'referrer' | 'referrerPolicy'
export type RequestOptions = Pick<RequestInit, 'signal'>

export class Problem extends Error {
  name = 'Problem'

  constructor(
    public message: string,
    public type: string,
    public title: string,
    public status: number,

    protected __raw?: Record<string, any>
  ) {
    super(message)
  }

  static fromResponse(resp: {
    response: Response
    error?: UnexpectedProblemResponse
  }): Problem {
    if (
      resp.response.headers.get('Content-Type') ===
        'application/problem+json' &&
      resp.error
    ) {
      return new Problem(
        resp.error.detail,
        resp.error.type,
        resp.error.title,
        resp.error.status ?? resp.response.status,
        resp.error
      )
    }

    return new Problem(
      `Request failed: ${resp.response.statusText}`,
      resp.response.statusText,
      resp.response.statusText,
      resp.response.status
    )
  }

  getField(key: string) {
    return this.__raw?.[key]
  }
}

// Implementation
export function transformResponse<
  T extends Record<string | number, any>,
  Options,
  Media extends MediaType,
>(
  resp: FetchResponse<T, Options, Media>
):
  | {
      data: ParseAsResponse<
        SuccessResponse<ResponseObjectMap<T>, Media>,
        Options
      >
      error?: never
      response: Response
    }
  | {
      data?: never
      error: Problem
      response: Response
    } {
  // Handle errors
  if (resp.error || resp.response.status >= 400) {
    const error = Problem.fromResponse(resp)

    return { error, response: resp.response }
  }

  // Decode dates
  if (resp.data) {
    resp.data = decodeDates(resp.data)
  }

  return { data: resp.data!, response: resp.response }
}

const ISODateFormat =
  /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d*)?(?:[-+]\d{2}:?\d{2}|Z)?$/

export function isIsoDateString(value: unknown): value is string {
  return typeof value === 'string' && ISODateFormat.test(value)
}

export function decodeDates<T>(data: T): T {
  // if it's a date string, return a date
  if (isIsoDateString(data)) {
    return new Date(data) as T
  }

  // if it's not an object or array, return it
  if (data === null || data === undefined || typeof data !== 'object') {
    return data
  }

  // if it's an array, decode each element
  if (Array.isArray(data)) {
    return data.map((val) => decodeDates(val)) as T
  }

  // if it's an object, decode each key
  for (const [key, val] of Object.entries(data)) {
    // @ts-expect-error we know this will give back the same type
    data[key] = decodeDates(val)
  }

  return data as T
}

export function encodeDates<T>(data: T): T {
  // if it's a date, return a date string
  if (data instanceof Date) {
    return data.toISOString() as T
  }

  // if it's not an object or array, return it
  if (data === null || data === undefined || typeof data !== 'object') {
    return data
  }

  // if it's an array, encode each element
  if (Array.isArray(data)) {
    return data.map((val) => encodeDates(val)) as T
  }

  // if it's an object, encode each key
  for (const [key, val] of Object.entries(data)) {
    // @ts-expect-error we know this will give back the same type
    data[key] = encodeDates(val)
  }

  return data as T
}
