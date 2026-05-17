import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CustomersClientContext extends Client {

}export interface CustomersClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCustomersClientContext(
  endpoint: string,
  options?: CustomersClientOptions,
): CustomersClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
