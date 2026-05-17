import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CurrenciesOperationsClientContext extends Client {

}export interface CurrenciesOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCurrenciesOperationsClientContext(
  endpoint: string,
  options?: CurrenciesOperationsClientOptions,
): CurrenciesOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
