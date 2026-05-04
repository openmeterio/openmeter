import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CustomerCreditBalancesOperationsClientContext extends Client {

}export interface CustomerCreditBalancesOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCustomerCreditBalancesOperationsClientContext(
  endpoint: string,
  options?: CustomerCreditBalancesOperationsClientOptions,
): CustomerCreditBalancesOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
