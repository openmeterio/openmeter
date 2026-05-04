import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CustomerBillingOperationsClientContext extends Client {

}export interface CustomerBillingOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCustomerBillingOperationsClientContext(
  endpoint: string,
  options?: CustomerBillingOperationsClientOptions,
): CustomerBillingOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
