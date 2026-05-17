import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CustomerCreditTransactionOperationsClientContext extends Client {

}export interface CustomerCreditTransactionOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCustomerCreditTransactionOperationsClientContext(
  endpoint: string,
  options?: CustomerCreditTransactionOperationsClientOptions,
): CustomerCreditTransactionOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
