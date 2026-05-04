import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface CustomerCreditGrantExternalSettlementOperationsClientContext extends Client {

}export interface CustomerCreditGrantExternalSettlementOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createCustomerCreditGrantExternalSettlementOperationsClientContext(
  endpoint: string,
  options?: CustomerCreditGrantExternalSettlementOperationsClientOptions,
): CustomerCreditGrantExternalSettlementOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
