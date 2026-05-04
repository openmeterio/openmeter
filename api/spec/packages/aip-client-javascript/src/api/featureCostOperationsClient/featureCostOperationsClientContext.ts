import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface FeatureCostOperationsClientContext extends Client {

}export interface FeatureCostOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createFeatureCostOperationsClientContext(
  endpoint: string,
  options?: FeatureCostOperationsClientOptions,
): FeatureCostOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
