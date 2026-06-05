import { type Options } from 'ky'

export const ServerList = [
  'https://{region}.api.konghq.com/v3',
  'http://localhost:{port}/api/v3',
  'https://openmeter.cloud/api/v3',
] as const

export const Regions = [
  'in',
  'me',
  'au',
  'eu',
  'us',
] as const

export type Region = (typeof Regions)[number]

export type ServerVariables = {
  region?: Region
  port?: string | number
}

export interface SDKOptions extends Omit<Options, 'method'> {
  baseUrl: (typeof ServerList)[number] | URL | string
  serverVariables?: ServerVariables
  apiKey?: string | (() => string | Promise<string>)
}
