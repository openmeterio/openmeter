import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface AppsOperationsClientContext extends Client {

}export interface AppsOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createAppsOperationsClientContext(
  endpoint: string,
  options?: AppsOperationsClientOptions,
): AppsOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
