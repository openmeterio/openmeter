import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface TaxClientContext extends Client {

}export interface TaxClientOptions extends ClientOptions {
  endpoint?: string;
}export function createTaxClientContext(
  endpoint: string,
  options?: TaxClientOptions,
): TaxClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
