import { type Options } from 'ky'

export const ServerList = [
  'https://{region}.api.konghq.com/v3',
  'http://localhost:{port}/api/v3',
  'https://openmeter.cloud/api/v3',
] as const

export const Regions = ['in', 'me', 'au', 'eu', 'us'] as const

export type Region = (typeof Regions)[number]

export type ServerVariables = {
  region?: Region
  port?: string | number
}

export interface SDKOptions extends Omit<Options, 'method'> {
  baseUrl: (typeof ServerList)[number] | URL | string
  serverVariables?: ServerVariables
  apiKey?: string | (() => string | Promise<string>)
  /**
   * Validate request bodies and response payloads against their schemas. Off by
   * default: the SDK maps casing but does not validate, so additive server fields
   * never break clients. When on, a request body or response that fails its schema
   * (missing/wrong-typed field, unknown enum value) returns a failed Result whose
   * `error` is a ValidationError (validation runs inside the SDK's request
   * handling, so it never rejects/throws).
   */
  validate?: boolean
}
