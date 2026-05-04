import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface LlmCostClientContext extends Client {

}export interface LlmCostClientOptions extends ClientOptions {
  endpoint?: string;
}export function createLlmCostClientContext(
  endpoint: string,
  options?: LlmCostClientOptions,
): LlmCostClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
