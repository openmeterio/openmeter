import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface BillingClientContext extends Client {

}export interface BillingClientOptions extends ClientOptions {
  endpoint?: string;
}export function createBillingClientContext(
  endpoint: string,
  options?: BillingClientOptions,
): BillingClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
