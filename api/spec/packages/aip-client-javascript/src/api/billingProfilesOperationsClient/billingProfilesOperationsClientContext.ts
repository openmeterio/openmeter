import {
  type Client,
  type ClientOptions,
  getClient,
} from "@typespec/ts-http-runtime";

export interface BillingProfilesOperationsClientContext extends Client {

}export interface BillingProfilesOperationsClientOptions extends ClientOptions {
  endpoint?: string;
}export function createBillingProfilesOperationsClientContext(
  endpoint: string,
  options?: BillingProfilesOperationsClientOptions,
): BillingProfilesOperationsClientContext {
  const params: Record<string, any> = {
    endpoint: endpoint
  };
  const resolvedEndpoint = "{endpoint}".replace(/{([^}]+)}/g, (_, key) =>
    key in params ? String(params[key]) : (() => { throw new Error(`Missing parameter: ${key}`); })()
  );;return getClient(resolvedEndpoint,{
    ...options
  })
}
