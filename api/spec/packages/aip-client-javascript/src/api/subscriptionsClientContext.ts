import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface SubscriptionsClientContext extends Client {

}export interface SubscriptionsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createSubscriptionsClientContext(
  endpoint: string,
  options?: SubscriptionsClientOptions,
): SubscriptionsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
