import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CustomerChargesOperationsClientContext extends Client {

}export interface CustomerChargesOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCustomerChargesOperationsClientContext(
  endpoint: string,
  options?: CustomerChargesOperationsClientOptions,
): CustomerChargesOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
