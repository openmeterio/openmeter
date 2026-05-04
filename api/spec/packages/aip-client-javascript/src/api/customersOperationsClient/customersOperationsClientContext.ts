import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CustomersOperationsClientContext extends Client {

}export interface CustomersOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCustomersOperationsClientContext(
  endpoint: string,
  options?: CustomersOperationsClientOptions,
): CustomersOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
