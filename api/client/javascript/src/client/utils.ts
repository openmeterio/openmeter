import type { FetchResponse, ParseAsResponse } from 'openapi-fetch'
import type {
  MediaType,
  ResponseObjectMap,
  SuccessResponse,
} from 'openapi-typescript-helpers'
import { HTTPError } from './common.js'

/**
 * Transform a response from the API
 * @param resp - The response to transform
 * @throws HTTPError if the response is an error
 * @returns The transformed response
 */
export function transformResponse<
  T extends Record<string | number, unknown>,
  Options,
  Media extends MediaType,
>(
  resp: FetchResponse<T, Options, Media>,
):
  | ParseAsResponse<SuccessResponse<ResponseObjectMap<T>, Media>, Options>
  | undefined
  | never {
  // Handle errors
  if (resp.error || resp.response.status >= 400) {
    throw HTTPError.fromResponse(resp)
  }

  // Decode dates
  resp.data = decodeDates(resp.data)
  return resp.data
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
