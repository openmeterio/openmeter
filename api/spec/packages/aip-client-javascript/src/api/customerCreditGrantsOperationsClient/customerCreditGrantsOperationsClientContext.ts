import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CustomerCreditGrantsOperationsClientContext extends Client {

}export interface CustomerCreditGrantsOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCustomerCreditGrantsOperationsClientContext(
  endpoint: string,
  options?: CustomerCreditGrantsOperationsClientOptions,
): CustomerCreditGrantsOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
