import { HTTPError as KyHTTPError } from 'ky'
import * as schemas from '../models/schemas.js'
import { HTTPError } from '../models/errors.js'

export async function toError(e: unknown): Promise<Error> {
  if (e instanceof KyHTTPError) {
    // ky consumes the body into e.data; e.response.json() would throw.
    const parsed = schemas.baseError.safeParse(e.data)
    const error = parsed.success ? parsed.data : undefined
    return HTTPError.fromResponse({ response: e.response, error })
  }
  if (e instanceof Error) {
    return e
  }
  return new Error(String(e))
}
