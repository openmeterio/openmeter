import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CustomerCreditAdjustmentsOperationsClientContext extends Client {

}export interface CustomerCreditAdjustmentsOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCustomerCreditAdjustmentsOperationsClientContext(
  endpoint: string,
  options?: CustomerCreditAdjustmentsOperationsClientOptions,
): CustomerCreditAdjustmentsOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
