import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface DefaultsClientContext extends Client {

}export interface DefaultsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createDefaultsClientContext(
  endpoint: string,
  options?: DefaultsClientOptions,
): DefaultsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
