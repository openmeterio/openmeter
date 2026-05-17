import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface TaxCodesOperationsClientContext extends Client {

}export interface TaxCodesOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createTaxCodesOperationsClientContext(
  endpoint: string,
  options?: TaxCodesOperationsClientOptions,
): TaxCodesOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
