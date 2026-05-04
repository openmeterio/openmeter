import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface FeaturesClientContext extends Client {

}export interface FeaturesClientOptions extends ClientOptions {
  endpoint?: string;
}export function createFeaturesClientContext(
  endpoint: string,
  options?: FeaturesClientOptions,
): FeaturesClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
