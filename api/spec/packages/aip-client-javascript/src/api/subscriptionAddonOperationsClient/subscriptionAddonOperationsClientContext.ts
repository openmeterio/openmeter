import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface SubscriptionAddonOperationsClientContext extends Client {

}export interface SubscriptionAddonOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createSubscriptionAddonOperationsClientContext(
  endpoint: string,
  options?: SubscriptionAddonOperationsClientOptions,
): SubscriptionAddonOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
