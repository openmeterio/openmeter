import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CustomerEntitlementsOperationsClientContext extends Client {

}export interface CustomerEntitlementsOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCustomerEntitlementsOperationsClientContext(
  endpoint: string,
  options?: CustomerEntitlementsOperationsClientOptions,
): CustomerEntitlementsOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
