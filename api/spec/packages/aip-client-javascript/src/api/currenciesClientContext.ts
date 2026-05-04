import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CurrenciesClientContext extends Client {

}export interface CurrenciesClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCurrenciesClientContext(
  endpoint: string,
  options?: CurrenciesClientOptions,
): CurrenciesClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
