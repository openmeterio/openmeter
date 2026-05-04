import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CurrenciesCustomCostBasesEndpointsClientContext extends Client {

}export interface CurrenciesCustomCostBasesEndpointsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCurrenciesCustomCostBasesEndpointsClientContext(
  endpoint: "https://global.api.konghq.com/v3" | "https://in.api.konghq.com/v3" | "https://me.api.konghq.com/v3" | "https://au.api.konghq.com/v3" | "https://eu.api.konghq.com/v3" | "https://us.api.konghq.com/v3" | string,
  options?: CurrenciesCustomCostBasesEndpointsClientOptions,
): CurrenciesCustomCostBasesEndpointsClientContext {
  const params: Record<string, any> = {
    endpoint: options?.endpoint ?? "https://global.api.konghq.com/v3"
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
