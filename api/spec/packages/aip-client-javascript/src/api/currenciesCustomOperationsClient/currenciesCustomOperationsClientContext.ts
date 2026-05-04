import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CurrenciesCustomOperationsClientContext extends Client {

}export interface CurrenciesCustomOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCurrenciesCustomOperationsClientContext(
  endpoint: string,
  options?: CurrenciesCustomOperationsClientOptions,
): CurrenciesCustomOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
