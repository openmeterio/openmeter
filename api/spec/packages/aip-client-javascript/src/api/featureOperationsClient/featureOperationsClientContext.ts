import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface FeatureOperationsClientContext extends Client {

}export interface FeatureOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createFeatureOperationsClientContext(
  endpoint: string,
  options?: FeatureOperationsClientOptions,
): FeatureOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
