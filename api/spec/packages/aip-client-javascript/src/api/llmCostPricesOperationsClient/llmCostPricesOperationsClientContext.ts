import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface LlmCostPricesOperationsClientContext extends Client {

}export interface LlmCostPricesOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createLlmCostPricesOperationsClientContext(
  endpoint: string,
  options?: LlmCostPricesOperationsClientOptions,
): LlmCostPricesOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
