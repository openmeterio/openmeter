import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface EntitlementsClientContext extends Client {

}export interface EntitlementsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createEntitlementsClientContext(
  endpoint: string,
  options?: EntitlementsClientOptions,
): EntitlementsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
