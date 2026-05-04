import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface SubscriptionsOperationsClientContext extends Client {

}export interface SubscriptionsOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createSubscriptionsOperationsClientContext(
  endpoint: string,
  options?: SubscriptionsOperationsClientOptions,
): SubscriptionsOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
