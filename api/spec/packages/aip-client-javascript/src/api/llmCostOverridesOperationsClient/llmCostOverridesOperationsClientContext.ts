import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface LlmCostOverridesOperationsClientContext extends Client {

}export interface LlmCostOverridesOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createLlmCostOverridesOperationsClientContext(
  endpoint: string,
  options?: LlmCostOverridesOperationsClientOptions,
): LlmCostOverridesOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
