import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface AppsClientContext extends Client {

}export interface AppsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createAppsClientContext(
  endpoint: string,
  options?: AppsClientOptions,
): AppsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
