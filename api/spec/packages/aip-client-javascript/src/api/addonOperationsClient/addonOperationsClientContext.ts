import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface AddonOperationsClientContext extends Client {

}export interface AddonOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createAddonOperationsClientContext(
  endpoint: string,
  options?: AddonOperationsClientOptions,
): AddonOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
