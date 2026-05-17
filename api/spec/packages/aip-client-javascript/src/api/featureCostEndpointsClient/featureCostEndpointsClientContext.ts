import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface FeatureCostEndpointsClientContext extends Client {

}export interface FeatureCostEndpointsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createFeatureCostEndpointsClientContext(
  endpoint: "https://global.api.konghq.com/v3" | "https://in.api.konghq.com/v3" | "https://me.api.konghq.com/v3" | "https://au.api.konghq.com/v3" | "https://eu.api.konghq.com/v3" | "https://us.api.konghq.com/v3" | string,
  options?: FeatureCostEndpointsClientOptions,
): FeatureCostEndpointsClientContext {
  const params: Record<string, any> = {
    endpoint: options?.endpoint ?? "https://global.api.konghq.com/v3"
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
